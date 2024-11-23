package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"event-reservation-api/middlewares"
	"event-reservation-api/models"
)

/*
Fetch the role name associated with a given role ID.

Arguments:

	ctx: The request context.
	pool: A connection pool to the database.
	roleID: The ID of the role to fetch.

Returns:

	roleName and error (nil if successful).
*/
func FetchRole(ctx context.Context, pool *pgxpool.Pool, roleID int) (string, error) {
	var roleName string

	// fetch the role name by ID
	query := "SELECT name FROM roles WHERE id = $1"
	err := pool.QueryRow(ctx, query, roleID).Scan(&roleName)
	if err != nil {
		return "", err // if the query fails, return error
	}

	return strings.ToUpper(roleName), nil
}

/*
Fetch the role id associated with a given role name.

Arguments:

	ctx: The request context.
	pool: A connection pool to the database.
	roleName: The name of the role to fetch.

Returns:

	roleId and error (nil if successful).
*/
func FetchRoleId(ctx context.Context, pool *pgxpool.Pool, roleName string) (int, error) {
	var roleId int

	// ensure case insensitivity
	roleName = strings.ToUpper(roleName)

	// fetch the role name by ID
	query := "SELECT id FROM roles WHERE name = $1"
	err := pool.QueryRow(ctx, query, roleName).Scan(&roleId)
	if err != nil {
		return -1, err // if the query fails, return error
	}

	return roleId, nil
}

/*
Check if a user with the same username already exists.

Arguments:

	ctx: The request context.
	pool: A connection pool to the database.
	username: The username to check for duplicates.
	exludeID: Address of the ID to exclude from the check, if nil every record
		will be checked.
*/
func checkDuplicateUsername(
	ctx context.Context,
	pool *pgxpool.Pool,
	username string,
	excludeID *string,
) error {
	var existingID string
	var err error

	if excludeID == nil {
		query := `
					SELECT id
					FROM users
					WHERE username = $1
					LIMIT 1
		`
		err = pool.QueryRow(ctx, query, username).Scan(&existingID)

	} else {
		query := `
					SELECT id
					FROM users
					WHERE username = $1 AND id != $2
					LIMIT 1
		`
		err = pool.QueryRow(ctx, query, username, *excludeID).Scan(&existingID)
	}

	// if record is found, it means that entry already exists
	if err == nil {
		return fmt.Errorf("Username '%s' is already taken", username)
	}
	if err == pgx.ErrNoRows {
		return nil
	}

	// in case of an unexpected error
	return fmt.Errorf("Error checking for duplicate username: %w", err)
}

/*
Retrieve all users.

Available only for admin users.

Arguments:

	pool: A connection pool to the database.

Returns:

	http.HandlerFunc: A function handler for fetching users.
*/
func GetUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the claims from the context
		claims, err := middlewares.GetClaimsFromContext(r.Context())
		if err != nil {
			http.Error(w, "Unauthorized: Missing or invalid token", http.StatusUnauthorized)
			return
		}

		// check users' role
		role, ok := claims["role"].(string)
		if !ok || role != "ADMIN" {
			http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
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
			http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// parse the rows into JSON response
		users := []map[string]interface{}{}
		for rows.Next() {
			var id int
			var name, surname, username, roleName, email string
			var lastLogin, createdAt time.Time

			if err := rows.Scan(&id, &name, &surname, &username, &email, &lastLogin, &createdAt, &roleName); err != nil {
				http.Error(w, "Failed to parse user data", http.StatusInternalServerError)
				return
			}

			users = append(users, map[string]interface{}{
				"id":          id,
				"name":        name,
				"surnamename": surname,
				"username":    username,
				"email":       email,
				"last_login":  lastLogin,
				"created_at":  createdAt,
				"role_name":   roleName,
			})
		}

		// set the headers and encode the reponse
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(users); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

/*
Retrieve a user by ID.

Available only for admin users.

Arguments:

	pool: A connection pool to the database.

Returns:

	http.HandlerFunc: A function handler for fetching a user by ID.
*/
func GetUserByIDHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the user id from the query parameter
		vars := mux.Vars(r)
		userId, ok := vars["id"]

		if !ok {
			http.Error(w, "User ID not provided", http.StatusBadRequest)
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
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Failed to parse user data", http.StatusInternalServerError)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// parsed response
		userResponse := map[string]interface{}{
			"id":         user.ID,
			"name":       user.Name,
			"surname":    user.Surname,
			"username":   user.Username,
			"email":      user.Email,
			"last_login": user.LastLogin,
			"created_at": user.CreatedAt,
			"is_active":  user.IsActive,
			"role_name":  roleName,
		}

		// encode response
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(userResponse)
		if err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

/*
Create a new user.

Available only for admin users.

Arguments:

	pool: A connection pool to the database.A

Returns:

	http.HandlerFunc: A function handler for creating a new user.
*/
func CreateUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// in case user already exists
		err := checkDuplicateUsername(r.Context(), pool, user.Username, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict) // 409 Conflict
			return
		}

		// hash the password
		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(user.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		// query template for inserting a user
		query := `INSERT INTO Users (name, surname, username,
                          email, last_login, created_at,
                          password_hash, role_id, is_active)
					VALUES ($1, $2, $3, $4, NOW(), NOW(), $5, $6, $7)`

		// fetch the role ID associated with the role name
		roleId, err := FetchRoleId(r.Context(), pool, user.RoleName)
		if err != nil {
			http.Error(w, "Failed to fetch user roles", http.StatusInternalServerError)
			return
		}

		// execute the query
		_, err = pool.Exec(
			r.Context(),
			query,
			// user table fields
			user.Name,
			user.Surname,
			user.Username,
			user.Email,
			// last_login
			// created_at
			passwordHash,
			roleId,
			user.IsActive,
		)
		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		// write response
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write([]byte(`{"message": "User created successfully"}`))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}
}

/*
Update existing

Available only for admin users.

Arguments:

	pool: A connection pool to the database.A

Returns:

	http.HandlerFunc: A function handler for updating a new user.
*/
func UpdateUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// parse the user id from the url
		vars := mux.Vars(r)
		userId, ok := vars["id"]

		if !ok {
			http.Error(w, "User ID not provided", http.StatusBadRequest)
			return
		}

		// struct for the request payload
		var user_request struct {
			Name     *string `json:"name"`
			Surname  *string `json:"surname"`
			Username *string `json:"username"`
			Email    *string `json:"email"`
			RoleName *string `json:"role_name"`
			IsActive *bool   `json:"is_active"`
		}

		// decode the body and parse the request
		if err := json.NewDecoder(r.Body).Decode(&user_request); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// building the update query
		query := `UPDATE users SET `
		args := []interface{}{}
		idx := 1

		// go over each field and build the query accordingly
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

			// check for duplicate username
			err := checkDuplicateUsername(r.Context(), pool, *user_request.Username, &userId)
			if err != nil {
				http.Error(w, err.Error(), http.StatusConflict) // 409 Conflict
				return
			}

			query += fmt.Sprintf("username = $%d, ", idx)
			args = append(args, *user_request.Surname)
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
			http.Error(w, "Failed to update user", http.StatusInternalServerError)
			return
		}

		// write response
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"message": "User updated successfully"}`))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}
}

/*
Delete user by ID.

Available only for admin users.

Arguments:

	pool: A connection pool to the database.A

Returns:

	http.HandlerFunc: A function handler for deleting a user.
*/
func DeleteUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// parse the user id from the url
		vars := mux.Vars(r)
		userId, ok := vars["id"]

		if !ok {
			http.Error(w, "User ID not provided", http.StatusBadRequest)
			return
		}

		// execute the query
		query := `DELETE FROM users WHERE id = $1`
		_, err := pool.Exec(r.Context(), query, userId)
		if err != nil {
			http.Error(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}

		// write response
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"message": "User deleted successfully"}`))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}
}
