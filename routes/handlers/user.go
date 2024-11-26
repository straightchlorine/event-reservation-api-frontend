package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"event-reservation-api/models"
)

/*
Retrieve all users.

Available only for admin users.
*/
func GetUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the claims from the context
		if !isAdmin(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		// fetch users, along with their roles from the database
		query := `SELECT
				u.id,
				u.name,
				u.surname,
				u.username,
				u.email,
				u.last_login,
				u.created_at,
				r.name AS role_name
			FROM users u
			JOIN roles r ON u.role_id = r.id
			ORDER BY u.id ASC`

		rows, err := pool.Query(r.Context(), query)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch users", err)
			return
		}
		defer rows.Close()

		// parse the rows into JSON response
		users := []map[string]interface{}{}
		for rows.Next() {
			var user models.User
			var roleName string
			err := rows.Scan(
				&user.ID,
				&user.Name,
				&user.Surname,
				&user.Username,
				&user.Email,
				&user.LastLogin,
				&user.CreatedAt,
				&roleName,
			)
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to parse user data", err)
				return
			}
			users = append(users, map[string]interface{}{
				"id":         user.ID,
				"name":       user.Name,
				"surname":    user.Surname,
				"username":   user.Username,
				"email":      user.Email,
				"last_login": user.LastLogin,
				"created_at": user.CreatedAt,
				"role_name":  roleName,
			})
		}

		// set the headers and encode the reponse
		writeJSONResponse(w, http.StatusOK, users)
	}
}

/*
Retrieve a user by ID.

Available only for admin users.
*/
func GetUserByIDHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		// parse the user id from the url
		vars := mux.Vars(r)
		userId, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "User ID not provided in the url", nil)
			return
		}

		// query to get all the information about the user
		query := `SELECT
				u.id,
				u.name,
				u.surname,
				u.username,
				u.email,
				u.last_login,
				u.created_at,
				u.is_active,
				r.name AS role_name
			FROM users u
			JOIN roles r ON u.role_id = r.id
			WHERE u.id = $1`

		var roleName string
		user := models.User{}

		// query the database and return any errors during the Scan
		row := pool.QueryRow(r.Context(), query, userId)
		err := row.Scan(
			&user.ID,
			&user.Name,
			&user.Surname,
			&user.Username,
			&user.Email,
			&user.LastLogin,
			&user.CreatedAt,
			&user.IsActive,
			&roleName,
		)

		if err == pgx.ErrNoRows {
			handleError(w, http.StatusNotFound, "User not found", nil)
			return
		}
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch user", err)
			return
		}

		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"id":         user.ID,
			"name":       user.Name,
			"surname":    user.Surname,
			"username":   user.Username,
			"email":      user.Email,
			"last_login": user.LastLogin,
			"created_at": user.CreatedAt,
			"is_active":  user.IsActive,
			"role_name":  roleName,
		})
	}
}

/*
Create a new user.

Available only for admin users.
*/
func CreateUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// check if the user is an admin
		if !isAdmin(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		var user struct {
			Name     string `json:"name"`
			Surname  string `json:"surname"`
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
			RoleName string `json:"role_name"`
			IsActive bool   `json:"is_active"`
		}

		// parse the json request
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			handleError(w, http.StatusBadRequest, "Invalid JSON input", err)
			return
		}

		// validate input fields
		if user.Username == "" || user.Password == "" || user.RoleName == "" {
			handleError(w, http.StatusBadRequest, "Missing required fields", nil)
			return
		}

		// in case user already exists
		if err := checkDuplicateUsername(r.Context(), pool, user.Username, nil); err != nil {
			handleError(w, http.StatusConflict, "Username already exists", err)
			return
		}

		// hash the password
		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(user.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to hash password", err)
			return
		}

		// fetch the role ID associated with the role name
		roleId, err := FetchRoleId(r.Context(), pool, user.RoleName)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch user roles", err)
			return
		}

		// query template for inserting a user
		var userId int
		query := `
			INSERT INTO Users
				(name, surname, username,
				email, last_login, created_at,
				password_hash, role_id, is_active)
			VALUES ($1, $2, $3, $4, NOW(), NOW(), $5, $6, $7)
			RETURNING id
		`

		// execute the query
		err = pool.QueryRow(
			r.Context(),
			query,
			user.Name,
			user.Surname,
			user.Username,
			user.Email,
			passwordHash,
			roleId,
			user.IsActive,
		).Scan(&userId)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to create user", err)
			return
		}

		writeJSONResponse(
			w,
			http.StatusCreated,
			map[string]interface{}{
				"message": "User created successfully",
				"userId":  userId,
			},
		)
	}
}

/*
Update existing user.

Available only for admin users.
*/
func UpdateUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		// parse the user id from the url
		vars := mux.Vars(r)
		userId, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "User ID not provided in the query", nil)
			return
		}

		// struct for the request payload
		var user_request struct {
			Name     *string `json:"name"`
			Surname  *string `json:"surname"`
			Username *string `json:"username"`
			Password *string `json:"password"`
			Email    *string `json:"email"`
			RoleName *string `json:"role_name"`
			IsActive *bool   `json:"is_active"`
		}

		// decode the body and parse the request
		if err := json.NewDecoder(r.Body).Decode(&user_request); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// check if the user exists
		if user_request.Username != nil {
			if err := checkDuplicateUsername(r.Context(), pool, *user_request.Username, &userId); err != nil {
				handleError(w, http.StatusConflict, "Username already exists", err)
				return
			}
		}

		// building the update query
		query := `UPDATE users SET `
		args := []interface{}{}
		idx := 1

		var hashedPassword *string

		if user_request.Name != nil {
			query += fmt.Sprintf("name = $%d, ", idx)
			args = append(args, *user_request.Name)
			idx++
		}
		if user_request.Surname != nil {
			query += fmt.Sprintf("surname = $%d, ", idx)
			args = append(args, *user_request.Surname)
			idx++
		}
		if user_request.Username != nil {
			query += fmt.Sprintf("username = $%d, ", idx)
			args = append(args, *user_request.Surname)
			idx++
		}
		if user_request.Password != nil {
			hashed, err := bcrypt.GenerateFromPassword(
				[]byte(*user_request.Password),
				bcrypt.DefaultCost,
			)
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to hash password", err)
				return
			}
			hashedStr := string(hashed)
			hashedPassword = &hashedStr

			query += fmt.Sprintf("password_hash = $%d, ", idx)
			args = append(args, *hashedPassword)
			idx++
		}
		if user_request.Email != nil {
			query += fmt.Sprintf("email = $%d, ", idx)
			args = append(args, *user_request.Email)
			idx++
		}
		if user_request.RoleName != nil {

			// fetch the role ID associated with the role name
			roleId, err := FetchRoleId(r.Context(), pool, *user_request.RoleName)
			if err != nil {
				http.Error(w, "Failed to fetch user roles", http.StatusInternalServerError)
				return
			}

			query += fmt.Sprintf("role_id = $%d, ", idx)
			args = append(args, roleId)
			idx++
		}
		if user_request.IsActive != nil {
			query += fmt.Sprintf("is_active = $%d, ", idx)
			args = append(args, *user_request.IsActive)
			idx++
		}

		// remove trailing comma and add where clause
		query = strings.TrimSuffix(query, ", ") + fmt.Sprintf(" WHERE id = $%d", idx)
		args = append(args, userId)

		// execute the query
		_, err := pool.Exec(r.Context(), query, args...)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to update user", err)
			return
		}

		// write response
		writeJSONResponse(
			w,
			http.StatusOK,
			map[string]string{"message": "User updated successfully"},
		)
	}
}

/*
Delete user by ID.

Available only for admin users.
*/
func DeleteUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		// parse the user id from the url
		vars := mux.Vars(r)
		userId, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "User ID not provided in the url", nil)
			return
		}

		// execute the query
		query := `DELETE FROM users WHERE id = $1`
		_, err := pool.Exec(r.Context(), query, userId)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to delete user", err)
			return
		}

		// write response
		writeJSONResponse(
			w,
			http.StatusOK,
			map[string]string{"message": "User deleted successfully"},
		)
	}
}
