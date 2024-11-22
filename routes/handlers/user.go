package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/middlewares"
	// "event-reservation-api/models"
)

func fetchRole(pool *pgxpool.Pool, ctx context.Context, id int) string {
	// Query to fetch the role name by id
	rows, err := pool.Query(ctx, "SELECT name FROM roles WHERE id=$1", id)
	if err != nil {
		// If there was an error with the query, return an empty string
		return ""
	}
	defer rows.Close()

	// Ensure that at least one row is returned
	if rows.Next() {
		var roleName string
		// Scan the role name into the roleName variable
		if err := rows.Scan(&roleName); err != nil {
			// If scanning fails, return an empty string
			return ""
		}
		// Return the role name
		return roleName
	}

	// If no rows were found for the provided id, return an empty string
	return ""
}

// GetUserHandler retrieves all users.
func GetUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract claims for role validation
		claims, err := middlewares.GetClaimsFromContext(r.Context())
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		role := claims["role"].(string)
		if strings.ToUpper(role) != "ADMIN" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		query := `SELECT u.id, u.password_hash, r.name AS role_name FROM users u JOIN roles r ON u.role_id = r.id`

		rows, err := pool.Query(r.Context(), query)
		if err != nil {
			http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []map[string]interface{}
		for rows.Next() {
			var user struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
				Role     string `json:"role"`
			}
			if err := rows.Scan(&user.ID, &user.Username, &user.Role); err != nil {
				http.Error(w, "Failed to parse user data", http.StatusInternalServerError)
				return
			}
			users = append(users, map[string]interface{}{
				"id":       user.ID,
				"username": user.Username,
				"role":     user.Role,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}
