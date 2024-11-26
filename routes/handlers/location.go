package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/models"
)

// Get all locations
func GetLocationsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `SELECT
				id,
				stadium,
				address,
				country,
				capacity
			FROM Locations
			ORDER BY stadium ASC`

		rows, err := pool.Query(r.Context(), query)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch locations", err)
			return
		}
		defer rows.Close()

		locations := []map[string]interface{}{}
		for rows.Next() {
			var location models.Location

			err := rows.Scan(
				&location.ID,
				&location.Stadium,
				&location.Address,
				&location.Country,
				&location.Capacity,
			)
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to parse location data", err)
				return
			}

			locations = append(locations, map[string]interface{}{
				"id":       location.ID,
				"stadium":  location.Stadium,
				"address":  location.Address,
				"country":  location.Country,
				"capacity": location.Capacity,
			})
		}

		writeJSONResponse(w, http.StatusOK, locations)
	}
}

// Retrieve a location by ID
func GetLocationByIDHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		locationID, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "Location ID not provided", nil)
			return
		}

		query := `SELECT
				id,
				stadium,
				address,
				country,
				capacity
			FROM Locations
			WHERE id = $1`

		var location models.Location
		row := pool.QueryRow(r.Context(), query, locationID)

		err := row.Scan(
			&location.ID,
			&location.Stadium,
			&location.Address,
			&location.Country,
			&location.Capacity,
		)
		if err == pgx.ErrNoRows {
			handleError(w, http.StatusNotFound, "Location not found", nil)
			return
		}
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch location", err)
			return
		}

		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"id":       location.ID,
			"stadium":  location.Stadium,
			"address":  location.Address,
			"country":  location.Country,
			"capacity": location.Capacity,
		})
	}
}

// Create a new location
func CreateLocationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the user is an admin
		if !isAdmin(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		var input struct {
			Stadium  string `json:"stadium"`
			Address  string `json:"address"`
			Country  string `json:"country,omitempty"`
			Capacity int    `json:"capacity"`
		}
		// Decode the input
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			handleError(w, http.StatusBadRequest, "Invalid JSON input", err)
			return
		}

		// Validate required fields
		if input.Stadium == "" || input.Address == "" || input.Capacity <= 0 {
			handleError(w, http.StatusBadRequest, "Missing or invalid fields", nil)
			return
		}

		// If country is not provided, set it to empty string
		if input.Country == "" {
			input.Country = ""
		}

		// Insert new location
		query := `
			INSERT INTO Locations (stadium, address, country, capacity)
			VALUES ($1, $2, $3, $4)
			RETURNING id`

		var locationID int
		err := pool.QueryRow(
			r.Context(),
			query,
			input.Stadium,
			input.Address,
			input.Country,
			input.Capacity,
		).Scan(&locationID)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to create location", err)
			return
		}

		writeJSONResponse(w, http.StatusCreated, map[string]interface{}{
			"message":    "Location created successfully",
			"locationId": locationID,
		})
	}
}

// Update an existing location
func UpdateLocationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the user is an admin
		if !isAdmin(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		// Parse the location ID from the URL
		vars := mux.Vars(r)
		locationID, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "Location ID not provided", nil)
			return
		}

		// Struct for the request payload
		var input struct {
			Stadium  *string `json:"stadium,omitempty"`
			Address  *string `json:"address,omitempty"`
			Country  *string `json:"country,omitempty"`
			Capacity *int    `json:"capacity,omitempty"`
		}

		// Decode the body and parse the request
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			handleError(w, http.StatusBadRequest, "Invalid JSON input", err)
			return
		}

		// Building the update query dynamically
		query := `UPDATE Locations SET `
		args := []interface{}{}
		idx := 1

		if input.Stadium != nil {
			query += fmt.Sprintf("stadium = $%d, ", idx)
			args = append(args, *input.Stadium)
			idx++
		}
		if input.Address != nil {
			query += fmt.Sprintf("address = $%d, ", idx)
			args = append(args, *input.Address)
			idx++
		}
		if input.Country != nil {
			query += fmt.Sprintf("country = $%d, ", idx)
			args = append(args, *input.Country)
			idx++
		}
		if input.Capacity != nil {
			query += fmt.Sprintf("capacity = $%d, ", idx)
			args = append(args, *input.Capacity)
			idx++
		}

		// If no fields to update
		if len(args) == 0 {
			handleError(w, http.StatusBadRequest, "No fields to update", nil)
			return
		}

		// Remove trailing comma and add where clause
		query = strings.TrimSuffix(query, ", ") + fmt.Sprintf(" WHERE id = $%d", idx)
		args = append(args, locationID)

		// Execute the query
		_, err := pool.Exec(r.Context(), query, args...)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to update location", err)
			return
		}

		// Write response
		writeJSONResponse(w, http.StatusOK, map[string]string{
			"message": "Location updated successfully",
		})
	}
}

// Delete a location by ID
func DeleteLocationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the user is an admin
		if !isAdmin(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		// Parse the location ID from the URL
		vars := mux.Vars(r)
		locationID, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "Location ID not provided", nil)
			return
		}

		// Execute the query
		query := `DELETE FROM Locations WHERE id = $1`
		_, err := pool.Exec(r.Context(), query, locationID)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to delete location", err)
			return
		}

		// Write response
		writeJSONResponse(w, http.StatusOK, map[string]string{
			"message": "Location deleted successfully",
		})
	}
}
