package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/middlewares"
)

// Utility to check if a location exists within the database.
func getLocationID(
	r *http.Request,
	w http.ResponseWriter,
	tx pgx.Tx,
	locationAddress *string,
	locationStadium *string,
	locationCapacity *int,
	locationCountry *string,
) (int, error) {
	// If location is nil, return an error (or handle it as needed)
	if locationAddress == nil || locationStadium == nil {
		return -1, fmt.Errorf("Insufficient location data: address and stadium are required")
	}

	var locationID int
	var err error

	// query to check if location exists
	checkIfLocationExists := `
		SELECT id
		FROM Locations
		WHERE address = $1 OR stadium = $2
	`

	// execute the query and scan the result
	err = tx.QueryRow(
		r.Context(),
		checkIfLocationExists,
		locationAddress,
		locationStadium,
	).Scan(&locationID)

	// if no rows are returned, the location does not exist
	if err != nil {
		if err == pgx.ErrNoRows {
			// insert new location
			locationInsertQuery := `
				INSERT INTO Locations (address, capacity, country, stadium)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`

			if locationCapacity == nil {
				*locationCapacity = 0
			}

			if locationCountry == nil {
				*locationCountry = ""
			}

			err = tx.QueryRow(
				r.Context(),
				locationInsertQuery,
				*locationAddress,
				*locationCapacity,
				*locationCountry,
				*locationStadium,
			).Scan(&locationID)
			if err != nil {
				handleError(
					w,
					http.StatusInternalServerError,
					"Failed to insert new location",
					err,
				)
				return -1, err
			}
		} else {
			handleError(w, http.StatusInternalServerError, "Failed to check existing location", err)
			return -1, err
		}
	}
	return locationID, nil
}

// Utilitity to convert a date string to RFC3339 format.
func dateToRFC3339(w http.ResponseWriter, date string) (string, error) {
	const customDateFormat = "2006-01-02 15:04"
	var parsedDate time.Time

	// Try parsing with the custom format
	parsedDate, err := time.Parse(customDateFormat, date)
	if err != nil {
		// If custom format fails, try RFC3339
		parsedDate, err = time.Parse(time.RFC3339, date)
		if err != nil {
			handleError(
				w,
				http.StatusBadRequest,
				"Invalid date format; must be YYYY-MM-DD HH:MM or RFC3339",
				err,
			)
			return "", err
		}
	}

	// Convert to RFC3339 for storage
	rfc3339Date := parsedDate.Format(time.RFC3339)
	return rfc3339Date, nil
}

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
