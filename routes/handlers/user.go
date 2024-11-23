package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/middlewares"
)

// FetchRole fetches the role name associated with a given role ID.

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

// GetUserHandler retrieves all users. Only accessible to users with the ADMIN role.

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
		query := `
			SELECT
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
