package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/models"
)

// Get all events
func GetEventsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `SELECT
				e.id,
				e.name,
				e.date,
				e.price,
				e.available_tickets,
				l.stadium,
				l.address,
				l.country,
				l.capacity
			FROM events e
			JOIN locations l ON e.location_id = l.id
			ORDER BY e.date ASC`

		rows, err := pool.Query(r.Context(), query)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch events", err)
			return
		}
		defer rows.Close()

		events := []map[string]interface{}{}
		for rows.Next() {
			var event models.Event
			var location models.Location

			err := rows.Scan(
				&event.ID,
				&event.Name,
				&event.Date,
				&event.Price,
				&event.AvailableTickets,
				&location.Stadium,
				&location.Address,
				&location.Country,
				&location.Capacity,
			)
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to parse event data", err)
				return
			}

			location_json := map[string]interface{}{
				"stadium":  location.Stadium,
				"address":  location.Address,
				"country":  location.Country,
				"capacity": location.Capacity,
			}

			events = append(events, map[string]interface{}{
				"id":                event.ID,
				"name":              event.Name,
				"price":             event.Price,
				"date":              event.Date,
				"available_tickets": event.AvailableTickets,
				"location":          location_json,
			})
		}

		writeJSONResponse(w, http.StatusOK, events)
	}
}

// Retrieve an event by ID.
func GetEventByIDHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventID, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "Event ID not provided", nil)
			return
		}

		query := `SELECT
				e.id,
				e.name,
				e.date,
				e.price,
				e.available_tickets,
				l.stadium,
				l.address,
				l.country,
				l.capacity
			FROM events e
			JOIN locations l ON e.location_id = l.id
			WHERE e.id = $1`

		row := pool.QueryRow(r.Context(), query, eventID)

		var event models.Event
		var location models.Location

		err := row.Scan(
			&event.ID,
			&event.Name,
			&event.Date,
			&event.Price,
			&event.AvailableTickets,
			&location.Stadium,
			&location.Address,
			&location.Country,
			&location.Capacity,
		)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to parse event data", err)
			return
		}

		location_json := map[string]interface{}{
			"stadium":  location.Stadium,
			"address":  location.Address,
			"country":  location.Country,
			"capacity": location.Capacity,
		}

		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"id":                event.ID,
			"name":              event.Name,
			"date":              event.Date,
			"price":             event.Price,
			"available_tickets": event.AvailableTickets,
			"location":          location_json,
		})
	}
}

// Create a new event
func CreateEventHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		// input structure in order to create an event
		var input struct {
			Name             string  `json:"name"`
			Date             string  `json:"date"`
			AvailableTickets int     `json:"available_tickets"`
			Price            float64 `json:"price"`
			Location         struct {
				Address  string `json:"address"`
				Capacity int    `json:"capacity,omitempty"`
				Country  string `json:"country,omitempty"`
				Stadium  string `json:"stadium"`
			} `json:"location"`
		}

		// decode the input
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			handleError(w, http.StatusBadRequest, "Invalid JSON input", err)
			return
		}

		// check if the required fields are present
		if input.Name == "" || input.Date == "" || input.Location.Address == "" ||
			input.AvailableTickets < 0 {
			handleError(w, http.StatusBadRequest, "Missing or invalid fields", nil)
			return
		}

		// convert date to rfc3339 format
		rfc3339Date, err := dateToRFC3339(w, input.Date)
		if err != nil && rfc3339Date == "" {
			return
		}

		// to ensure atomicity, we will start a transaction
		tx, err := pool.Begin(r.Context())
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to start transaction", err)
			return
		}
		defer tx.Rollback(r.Context())

		locationID, err := getLocationID(
			r,
			w,
			tx,
			&input.Location.Address,
			&input.Location.Stadium,
			&input.Location.Capacity,
			&input.Location.Country,
		)
		if err != nil {
			return
		}

		// insert new event
		var eventID int
		eventQuery := `
				INSERT INTO Events (name, date, price, location_id, available_tickets)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
		`
		err = tx.QueryRow(
			r.Context(),
			eventQuery,
			input.Name,
			rfc3339Date,
			input.Price,
			locationID,
			input.AvailableTickets,
		).Scan(&eventID)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to create event", err)
			return
		}

		// commit the transaction
		if err = tx.Commit(r.Context()); err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to commit transaction", err)
			return
		}

		writeJSONResponse(w, http.StatusCreated, map[string]interface{}{
			"message": "Event created successfully",
			"eventId": eventID,
		})
	}
}

// Update an existing event
func UpdateEventHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventId, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "Invalid event ID", nil)
			return
		}

		var input struct {
			AvailableTickets *int     `json:"available_tickets,omitempty"`
			Date             *string  `json:"date,omitempty"`
			Name             *string  `json:"name,omitempty"`
			Price            *float64 `json:"price"`
			Location         *struct {
				Address  string `json:"address,omitempty"`
				Capacity int    `json:"capacity,omitempty"`
				Country  string `json:"country,omitempty"`
				Stadium  string `json:"stadium,omitempty"`
			} `json:"location,omitempty"`
		}

		// parse the input
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			handleError(w, http.StatusBadRequest, "Invalid JSON input", err)
			return
		}

		// ensure atomicity
		tx, err := pool.Begin(r.Context())
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to start transaction", err)
			return
		}
		defer tx.Rollback(r.Context())

		var updateQueries []string
		var updateArgs []interface{}
		argIndex := 1

		// update event fields dynamically based on provided input
		if input.Name != nil {
			updateQueries = append(updateQueries, fmt.Sprintf("name = $%d", argIndex))
			updateArgs = append(updateArgs, *input.Name)
			argIndex++
		}

		if input.AvailableTickets != nil {
			updateQueries = append(updateQueries, fmt.Sprintf("available_tickets = $%d", argIndex))
			updateArgs = append(updateArgs, *input.AvailableTickets)
			argIndex++
		}

		if input.Date != nil {
			rfc3339Date, err := dateToRFC3339(w, *input.Date)
			if err != nil && rfc3339Date == "" {
				return
			}

			updateQueries = append(updateQueries, fmt.Sprintf("date = $%d", argIndex))
			updateArgs = append(updateArgs, rfc3339Date)
			argIndex++
		}

		if input.Location != nil {
			locationID, err := getLocationID(
				r,
				w,
				tx,
				&input.Location.Address,
				&input.Location.Stadium,
				&input.Location.Capacity,
				&input.Location.Country,
			)
			if err != nil {
				return
			}

			updateQueries = append(updateQueries, fmt.Sprintf("location_id = $%d", argIndex))
			updateArgs = append(updateArgs, locationID)
			argIndex++
		}
		if input.Price != nil {
			updateQueries = append(updateQueries, fmt.Sprintf("price = $%d", argIndex))
			updateArgs = append(updateArgs, *input.Price)
			argIndex++
		}

		if len(updateQueries) == 0 {
			handleError(w, http.StatusBadRequest, "No fields to update", nil)
			return
		}

		updateArgs = append(updateArgs, eventId)
		updateQuery := fmt.Sprintf(
			"UPDATE Events SET %s WHERE id = $%d",
			strings.Join(updateQueries, ", "),
			argIndex,
		)

		_, err = tx.Exec(r.Context(), updateQuery, updateArgs...)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to update event", err)
			return
		}

		if err = tx.Commit(r.Context()); err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to commit transaction", err)
			return
		}

		writeJSONResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Event updated successfully",
		})
	}
}

// Delete an event by ID
func DeleteEventHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventID, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "Event ID not provided", nil)
			return
		}

		query := `DELETE FROM Events WHERE id = $1`

		_, err := pool.Exec(r.Context(), query, eventID)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to delete event", err)
			return
		}

		writeJSONResponse(w, http.StatusOK, map[string]string{
			"message": "Event deleted successfully",
		})
	}
}
