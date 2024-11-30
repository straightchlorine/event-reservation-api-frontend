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

// GetLocationsHandler lists all locations from the database
//
//	@Summary		Get all locations
//	@Description	Retrieve a list of all locations.
//	@ID				api.getLocations
//	@Tags			locations
//	@Produce		json
//	@Success		200	{object}	models.LocationsResponse	"List of locations"
//	@Failure		500	{object}	models.ErrorResponse		"Internal Server Error"
//	@Failure		404	{object}	models.ErrorResponse		"Not Found"
//	@Router			/locations [get]
func GetLocationsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
			SELECT
				id, stadium, address, country, capacity
			FROM Locations
			ORDER BY id ASC
		`

		// execute the query
		rows, err := pool.Query(r.Context(), query)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch locations.")
			return
		}
		defer rows.Close()

		// build the list of locations
		locations := []models.LocationResponse{}
		for rows.Next() {
			var location models.LocationResponse

			err := rows.Scan(
				&location.ID,
				&location.Stadium,
				&location.Address,
				&location.Country,
				&location.Capacity,
			)
			if err != nil {
				if err == pgx.ErrNoRows {
					writeErrorResponse(
						w,
						http.StatusNotFound,
						"No locations found in the database.",
					)
				}
				writeErrorResponse(
					w,
					http.StatusInternalServerError,
					"Failed to parse location data.",
				)
				return
			}
			locations = append(locations, location)
		}
		locations_response := models.LocationsResponse{Locations: locations}
		writeJSONResponse(w, http.StatusOK, locations_response)
	}
}

// GetLocationByIDHandler returns a single location by ID.
//
//	@Summary		Retrieve a location.
//	@Description	Retrieve a single location by ID.
//	@ID				api.getLocation
//	@Tags			locations
//	@Produce		json
//	@Param			id	path		string					true	"Location ID"
//	@Success		200	{object}	models.LocationResponse	"Location details"
//	@Failure		500	{object}	models.ErrorResponse	"Internal Server Error"
//	@Failure		404	{object}	models.ErrorResponse	"Not Found"
//	@Router			/locations/{id} [get]
func GetLocationByIDHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		locationID, ok := vars["id"]
		if !ok {
			writeErrorResponse(w, http.StatusBadRequest, "Location ID not provided in the URL.")
			return
		}

		query := `
			SELECT id, stadium, address, country, capacity
			FROM Locations
			WHERE id = $1
		`

		var location models.LocationResponse
		row := pool.QueryRow(r.Context(), query, locationID)

		if err := row.Scan(
			&location.ID,
			&location.Stadium,
			&location.Address,
			&location.Country,
			&location.Capacity,
		); err != nil {
			if err == pgx.ErrNoRows {
				writeErrorResponse(w, http.StatusNotFound, "Location not found.")
				return
			}
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch the location.")
			return
		}
		writeJSONResponse(w, http.StatusOK, location)
	}
}

// CreateLocationHandler creates a single location in the database.
//
//	@Summary		Create a new location.
//	@Description	Parse the payload and create a new location with provided dataset.
//	@ID				api.createLocation
//	@Tags			locations
//	@Produce		json
//	@Accept			json
//	@Param			body	body		models.CreateLocationRequest	true	"Payload to create a location"
//	@Success		200		{object}	models.SuccessResponseCreate	"Location created successfully"
//	@Failure		400		{object}	models.ErrorResponse			"Bad Request"
//	@Failure		403		{object}	models.ErrorResponse			"Forbidden"
//	@Failure		500		{object}	models.ErrorResponse			"Internal Server Error"
//	@Router			/locations [put]
func CreateLocationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			writeErrorResponse(
				w,
				http.StatusForbidden,
				"Insufficient permissions to create a location.",
			)
			return
		}

		// decode the request body
		input := models.CreateLocationRequest{}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON input.")
			return
		}

		// validate fields
		if input.Stadium == "" || input.Address == "" || input.Capacity <= 0 {
			writeErrorResponse(
				w,
				http.StatusBadRequest,
				"Missing or invalid fields in the payload.",
			)
			return
		}

		// set default value for country
		if input.Country == "" {
			input.Country = "N/A"
		}

		// execute the query
		var locationID int
		query := `
			INSERT INTO Locations (stadium, address, country, capacity)
			VALUES ($1, $2, $3, $4)
			RETURNING id`
		if err := pool.QueryRow(
			r.Context(),
			query,
			input.Stadium,
			input.Address,
			input.Country,
			input.Capacity,
		).Scan(&locationID); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to create a location.")
			return
		}

		writeJSONResponse(
			w,
			http.StatusCreated,
			models.SuccessResponseCreate{Message: "Location created successfully", ID: locationID},
		)
	}
}

// UpdateLocationHandler updates an existing location by ID.
//
//	@Summary		Update an existing location
//	@Description	Update location details based on the provided payload.
//	@ID				api.updateLocation
//	@Tags			locations
//	@Produce		json
//	@Accept			json
//	@Param			id		path		string							true	"Location ID"
//	@Param			body	body		models.UpdateLocationRequest	true	"Payload to update a location"
//	@Success		200		{object}	models.SuccessResponse			"Event updated successfully"
//	@Failure		400		{object}	models.ErrorResponse			"Bad Request"
//	@Failure		403		{object}	models.ErrorResponse			"Forbidden"
//	@Failure		422		{object}	models.ErrorResponse			"Unprocessable Entity"
//	@Failure		500		{object}	models.ErrorResponse			"Internal Server Error"
//	@Router			/locations/{id} [put]
func UpdateLocationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// check if the user is an admin
		if !isAdmin(r) {
			writeErrorResponse(
				w,
				http.StatusForbidden,
				"Insufficient permissions to update a location.",
			)
			return
		}

		// parse the location ID from the URL
		vars := mux.Vars(r)
		locationID, ok := vars["id"]
		if !ok {
			writeErrorResponse(w, http.StatusBadRequest, "Location ID not provided.")
			return
		}

		// decode the body and parse the request
		input := models.UpdateLocationRequest{}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON input.")
			return
		}

		// building the update query dynamically
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

		if len(args) == 0 {
			writeErrorResponse(w, http.StatusUnprocessableEntity, "Nothing to update.")
			return
		}

		query = strings.TrimSuffix(query, ", ") + fmt.Sprintf(" WHERE id = $%d", idx)
		args = append(args, locationID)

		_, err := pool.Exec(r.Context(), query, args...)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to update the location.")
			return
		}

		writeJSONResponse(
			w,
			http.StatusOK,
			models.SuccessResponse{Message: "Location updated successfully"},
		)
	}
}

// DeleteLocationHandler deletes an existing location by ID.
//
//	@Summary		Delete an existing location
//	@Description	Delete a location by its ID.
//	@ID				api.deleteLocation
//	@Tags			locations
//	@Produce		json
//	@Param			id	path		string							true	"Location ID"
//	@Success		200	{object}	models.SuccessResponseCreate	"Event deleted successfully"
//	@Failure		400	{object}	models.ErrorResponse			"Bad Request"
//	@Failure		500	{object}	models.ErrorResponse			"Internal Server Error"
//	@Router			/locations/{id} [delete]
func DeleteLocationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			writeErrorResponse(w, http.StatusForbidden, "Forbidden: Insufficient permissions")
			return
		}

		// parse the id
		vars := mux.Vars(r)
		locationID, ok := vars["id"]
		if !ok {
			writeErrorResponse(w, http.StatusBadRequest, "Location ID not provided.")
			return
		}

		// delete the user
		query := `DELETE FROM Locations WHERE id = $1`
		_, err := pool.Exec(r.Context(), query, locationID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete the location.")
			return
		}

		writeJSONResponse(
			w,
			http.StatusOK,
			models.SuccessResponse{Message: "Location deleted successfully"},
		)
	}
}
