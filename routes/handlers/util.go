package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/middlewares"
)

// Utility function to handle JSON responses.
func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Utility function for error handling.
func handleError(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		http.Error(w, fmt.Sprintf("%s: %v", message, err), status)
	} else {
		http.Error(w, message, status)
	}
}

// Helper function to check admin permissions.
func isAdmin(r *http.Request) bool {
	claims, err := middlewares.GetClaimsFromContext(r.Context())
	if err != nil {
		return false
	}
	role, ok := claims["role"].(string)
	return ok && role == "ADMIN"
}

// Helper function to check registered user permissions.
func isRegistered(r *http.Request) bool {
	claims, err := middlewares.GetClaimsFromContext(r.Context())
	if err != nil {
		return false
	}
	role, ok := claims["role"].(string)
	return ok && role == "REGISTERED"
}

// Fetch the role name associated with a given role ID.
func FetchRole(ctx context.Context, pool *pgxpool.Pool, roleID int) (string, error) {
	var roleName string
	query := "SELECT name FROM roles WHERE id = $1"
	err := pool.QueryRow(ctx, query, roleID).Scan(&roleName)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(roleName), nil
}

// Fetch the role ID associated with a given role name.
func FetchRoleId(ctx context.Context, pool *pgxpool.Pool, roleName string) (int, error) {
	var roleId int
	query := "SELECT id FROM roles WHERE name = $1"
	err := pool.QueryRow(ctx, query, strings.ToUpper(roleName)).Scan(&roleId)
	if err != nil {
		return -1, err
	}
	return roleId, nil
}

// Check if a username already exists, excluding a specific ID if provided.
func checkDuplicateUsername(
	ctx context.Context,
	pool *pgxpool.Pool,
	username string,
	excludeID *string,
) error {
	var query string
	var args []interface{}
	if excludeID == nil {
		query = "SELECT id FROM users WHERE username = $1 LIMIT 1"
		args = append(args, username)
	} else {
		query = "SELECT id FROM users WHERE username = $1 AND id != $2 LIMIT 1"
		args = append(args, username, *excludeID)
	}
	var existingID string
	err := pool.QueryRow(ctx, query, args...).Scan(&existingID)
	if err == nil {
		return fmt.Errorf("username '%s' is already taken", username)
	}
	if err == pgx.ErrNoRows {
		return nil
	}
	return fmt.Errorf("error checking for duplicate username: %w", err)
}
