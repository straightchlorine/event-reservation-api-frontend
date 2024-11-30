package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"event-reservation-api/models"
)

// GetUserHandler lists all users.
//
//	@Summary		List all users (admin only)
//	@Description	Retrieve a list of all users, including their details and roles.
//	@Tags			users
//	@Produce		json
//	@Success		200	{array}		models.UserResponse		"List of users"
//	@Failure		403	{object}	models.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	models.ErrorResponse	"Not Found"
//	@Failure		500	{object}	models.ErrorResponse	"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/users [get]
func GetUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// in order to see users, one must be an admin
		if !isAdmin(r) {
			writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions.")
			return
		}

		// fetch users and role names
		query := `
			SELECT u.id, u.name, u.surname, u.username, u.email,
				u.last_login, u.created_at, u.is_active,
				r.name as role_name
			FROM users u
			JOIN roles r ON u.role_id = r.id
			ORDER BY u.id ASC
		`
		rows, err := pool.Query(r.Context(), query)
		if err != nil {
			if err == pgx.ErrNoRows {
				writeErrorResponse(w, http.StatusNotFound, "No users in the database.")
				return
			}
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch users.")
			return
		}
		defer rows.Close()

		users := []models.UserResponse{}
		for rows.Next() {
			var user models.UserResponse

			if err := rows.Scan(
				&user.ID, &user.Name, &user.Surname, &user.Username, &user.Email,
				&user.LastLogin, &user.CreatedAt,
				&user.IsActive, &user.RoleName,
			); err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to parse user data.")
				return
			}

			users = append(users, user)
		}
		users_response := models.UsersResponse{Users: users}
		writeJSONResponse(w, http.StatusOK, users_response)
	}
}

// GetUserByIDHandler returns a single user by ID.
//
//	@Summary		Get a user by ID (admin only)
//	@Description	Retrieve a user, including its details and roles.
//	@Tags			users
//	@Produce		json
//	@Param			id	path		string					true	"User ID"
//	@Success		200	{object}	models.UserResponse		"User details"
//	@Failure		403	{object}	models.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	models.ErrorResponse	"Not Found"
//	@Failure		500	{object}	models.ErrorResponse	"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/users/{id} [get]
func GetUserByIDHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions.")
			return
		}

		userId, err := parseUserIdFromURL(r)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "User ID not provided in the URL.")
			return
		}

		// find user and its details with the given ID
		query := `
			SELECT
				u.id, u.name, u.surname, u.username, u.email,
				u.last_login, u.created_at, u.is_active,
				r.name
			FROM users u
			JOIN roles r ON u.role_id = r.id
			WHERE u.id = $1
		`
		user := models.UserResponse{}
		row := pool.QueryRow(r.Context(), query, userId)
		if err := row.Scan(
			&user.ID, &user.Name, &user.Surname, &user.Username, &user.Email,
			&user.LastLogin, &user.CreatedAt,
			&user.IsActive, &user.RoleName,
		); err != nil {
			if err == pgx.ErrNoRows {
				writeErrorResponse(w, http.StatusNotFound, "User not found.")
				return
			}
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to parse user data.")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// return the user
		writeJSONResponse(w, http.StatusOK, user)
	}
}

// CreateUserHandler creates a single user in the database.
//
//	@Summary		Create a new user.
//	@Description	Retrieve a user, including its details and roles.
//	@Tags			users
//	@Produce		json
//	@Param			id	path		string								true	"User ID"
//	@Success		200	{object}	models.SuccessResponseCreateUUID	"User details"
//	@Failure		403	{object}	models.ErrorResponse				"Forbidden"
//	@Failure		404	{object}	models.ErrorResponse				"Not Found"
//	@Failure		500	{object}	models.ErrorResponse				"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/users [put]
func CreateUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isAdmin := isAdmin(r)

		// parse the json request
		user := models.CreateUserRequest{}
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON input.")
			return
		}

		// check if permissions are sufficient and mandatory fields present
		status, err := validateCreateUserPayload(isAdmin, user)
		if status != 200 && err != nil {
			writeErrorResponse(w, status, err.Error())
		}

		// check if the username is unique
		status, err = isDuplicate(r.Context(), pool, user.Username)
		if err != nil {
			writeErrorResponse(w, status, err.Error())
			return
		}

		// hash the password
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to hash the password.")
			return
		}

		// fetch the role id, based on its name
		roleId, err := fetchRoleId(r.Context(), pool, user.RoleName)
		if err != nil {
			writeErrorResponse(w, roleId, err.Error())
		}

		// insert a new user
		var userId string
		query := `
			INSERT INTO users
				(name, surname, username, email, is_active, password_hash, role_id, last_login)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
			RETURNING id
		`
		if err := pool.QueryRow(
			r.Context(), query,
			user.Name,
			user.Surname,
			user.Username,
			user.Email,
			user.IsActive,
			passwordHash,
			roleId,
		).Scan(&userId); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to create the user.")
			return
		}

		writeJSONResponse(
			w,
			http.StatusCreated,
			models.SuccessResponseCreateUUID{
				Message: "User created successfully",
				UUID:    userId,
			},
		)
	}
}

// UpdateUserHandler updates a single user.
//
//	@Summary		Update user.
//	@Description	Update user details (only owner/admin).
//	@Tags			users
//	@Param			id	path		string					true	"User ID"
//	@Success		200	{object}	models.SuccessResponse	"User updated successfully"
//	@Failure		403	{object}	models.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	models.ErrorResponse	"Not Found"
//	@Failure		500	{object}	models.ErrorResponse	"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/users/{id} [put]
func UpdateUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, err := parseUserIdFromURL(r)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "User ID not provided in the URL.")
			return
		}

		if !isAdmin(r) && !isOwner(r, userId) {
			writeErrorResponse(
				w,
				http.StatusBadRequest,
				"Insufficient permissions to update selected user.",
			)
			return
		}

		// decode the body and parse the request
		req := models.UpdateUserRequest{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload.")
			return
		}

		// query starting point
		query := `UPDATE users SET `

		// arguments for filling the query
		args := []interface{}{}

		// index of each argument
		idx := 1

		// based on the present parameters, build update query
		if req.Username != nil {
			if status, err := isDuplicateExcept(r.Context(), pool, *req.Username, userId); err != nil {
				writeErrorResponse(w, status, err.Error())
				return
			}
			query += fmt.Sprintf("username = $%d, ", idx)
			args = append(args, *req.Username)
			idx++
		}
		if req.Name != nil {
			query += fmt.Sprintf("name = $%d, ", idx)
			args = append(args, *req.Name)
			idx++
		}
		if req.Surname != nil {
			query += fmt.Sprintf("surname = $%d, ", idx)
			args = append(args, *req.Surname)
			idx++
		}
		if req.Password != nil {
			hashed, err := bcrypt.GenerateFromPassword(
				[]byte(*req.Password),
				bcrypt.DefaultCost,
			)
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to hash password.")
				return
			}

			hashedStr := string(hashed)
			req.Password = &hashedStr

			query += fmt.Sprintf("password_hash = $%d, ", idx)
			args = append(args, *req.Password)
			idx++
		}
		if req.Email != nil {
			query += fmt.Sprintf("email = $%d, ", idx)
			args = append(args, *req.Email)
			idx++
		}
		if req.RoleName != nil {
			roleId, err := fetchRoleId(r.Context(), pool, *req.RoleName)
			if err != nil {
				writeErrorResponse(w, roleId, err.Error())
				return
			}

			query += fmt.Sprintf("role_id = $%d, ", idx)
			args = append(args, roleId)
			idx++
		}
		if req.IsActive != nil {
			query += fmt.Sprintf("is_active = $%d, ", idx)
			args = append(args, *req.IsActive)
			idx++
		}

		// remove trailing comma and add where clause
		query = strings.TrimSuffix(query, ", ") + fmt.Sprintf(" WHERE id = $%d", idx)
		args = append(args, userId)

		// update the user
		if _, err := pool.Exec(
			r.Context(),
			query,
			args...,
		); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to update user.")
			return
		}

		writeJSONResponse(
			w,
			http.StatusOK,
			models.SuccessResponse{Message: "User updated successfully."},
		)
	}
}

// DeleteUserHandler deletes specified user
//
//	@Summary		Delete user.
//	@Description	Deletes user from the database (only owner/admin).
//	@Tags			users
//	@Param			id	path		string					true	"User ID"
//	@Success		200	{object}	models.SuccessResponse	"User details"
//	@Failure		403	{object}	models.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	models.ErrorResponse	"Not Found"
//	@Failure		500	{object}	models.ErrorResponse	"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/users/{id} [delete]
func DeleteUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, err := parseUserIdFromURL(r)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "User ID not provided in the URL.")
			return
		}

		// only available for admins and owners, just as the update
		if !isAdmin(r) && !isOwner(r, userId) {
			writeErrorResponse(
				w,
				http.StatusBadRequest,
				"Insufficient permissions to delete selected user.",
			)
			return
		}

		// delete the user
		query := `DELETE FROM users WHERE id = $1`
		if _, err = pool.Exec(
			r.Context(),
			query,
			userId,
		); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete user.")
			return
		}

		writeJSONResponse(
			w,
			http.StatusOK,
			models.SuccessResponse{Message: "User deleted successfully"},
		)
	}
}
