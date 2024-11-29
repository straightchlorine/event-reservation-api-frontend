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

// GetEventsHandler lists all events in the database.
//
//	@Summary				Get all events
//	@Description		Retrieve a list of all available events with their details and locations.
//	@ID							api.getEvents
//	@Tags						events
//	@Produce				json
//	@Success				200	{array}		models.EventResponse	"List of events"
//	@Failure				500	{object}	models.ErrorResponse	"Internal Server Error"
//	@Failure				404	{object}	models.ErrorResponse	"Not Found"
//	@Router					/events [get]
func GetEventsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
			SELECT
				e.id, e.name, e.date, e.price, e.available_tickets,
				l.id, l.stadium, l.address, l.country, l.capacity
			FROM events e
			JOIN locations l ON e.location_id = l.id
			ORDER BY e.date ASC
		`

		// query the database for events
		rows, err := pool.Query(r.Context(), query)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch events.")
			return
		}
		defer rows.Close()

		// results in event and attached location information
		events := []models.EventResponse{}
		for rows.Next() {
			var event models.EventResponse
			var location models.LocationResponse

			if err := rows.Scan(
				&event.ID,
				&event.Name,
				&event.Date,
				&event.Price,
				&event.AvailableTickets,
				&location.ID,
				&location.Stadium,
				&location.Address,
				&location.Country,
				&location.Capacity,
			); err != nil {
				if err == pgx.ErrNoRows {
					writeErrorResponse(w, http.StatusNotFound, "No events in the database.")
					return
				}
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to parse events.")
				return
			}

			// append the location and event to the list
			event.Location = location
			events = append(events, event)
		}
		writeJSONResponse(w, http.StatusOK, events)
	}
}

// GetEventByIDHandler returns a single event by ID.
//
//	@Summary			Get an event by ID
//	@Description	Retrieve a list of all available events with their details and locations.
//	@ID						api.getEventByID
//	@Tags					events
//	@Produce			json
//	@Param				id			path		string								true	"Event ID"
//	@Success			200		{array}		models.EventResponse	"List of events"
//	@Failure			500		{object}	models.ErrorResponse	"Internal Server Error"
//	@Failure			404		{object}	models.ErrorResponse	"Not Found"
//	@Router				/events/{id} [get]
func GetEventByIDHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// parse the event id from the url
		vars := mux.Vars(r)
		eventID, ok := vars["id"]
		if !ok {
			writeErrorResponse(w, http.StatusBadRequest, "Event ID not provided in the URL.")
			return
		}

		// execute the query
		query := `
			SELECT
				e.id, e.name, e.date, e.price, e.available_tickets,
				l.id, l.stadium, l.address, l.country, l.capacity
			FROM events e
			JOIN locations l ON e.location_id = l.id
			WHERE e.id = $1
		`
		row := pool.QueryRow(r.Context(), query, eventID)

		// build the structs
		var event models.EventResponse
		var location models.LocationResponse
		if err := row.Scan(
			&event.ID,
			&event.Name,
			&event.Date,
			&event.Price,
			&event.AvailableTickets,
			&location.ID,
			&location.Stadium,
			&location.Address,
			&location.Country,
			&location.Capacity,
		); err != nil {
			if err == pgx.ErrNoRows {
				writeErrorResponse(w, http.StatusNotFound, "Event not found.")
				return
			}
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to parse selected event.")
			return
		}

		event.Location = location
		writeJSONResponse(w, http.StatusOK, event)
	}
}

// CreateEventHandler creates a single event in the database.
//
//	@Summary			Create a new event.
//	@Description	Parse the payload and create a new event with provided dataset.
//	@ID						api.createEvent
//	@Tags					events
//	@Produce			json
//	@Accept				json
//	@Param				body		body		models.CreateEventRequest			true	"Payload to create an event"
//	@Success			200		{object}	models.SuccessResponseCreate	"Event created successfully"
//	@Failure			400		{object}	models.ErrorResponse					"Bad Request"
//	@Failure			500		{object}	models.ErrorResponse					"Internal Server Error"
//	@Router				/events [put]
func CreateEventHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions.")
			return
		}

		// event structure in order to create an event
		event := models.CreateEventRequest{}

		// decode the input
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload.")
			return
		}

		// check if the required fields are present
		if event.Name == "" || event.Date == "" || event.Location.Address == "" ||
			event.AvailableTickets < 0 {
			writeErrorResponse(w, http.StatusBadRequest, "Missing or invalid fields.")
			return
		}

		// convert date to rfc3339 or yyyy-mm-dd hh:mm format
		rfc3339Date, err := dateToRFC3339(w, event.Date)
		if err != nil && rfc3339Date == "" {
			writeErrorResponse(
				w,
				http.StatusBadRequest,
				"Invalid date format; must be YYYY-MM-DD HH:MM or RFC3339.",
			)
			return
		}

		// to ensure atomicity, we will start a transaction
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction.")
			return
		}
		defer tx.Rollback(r.Context())

		// check if location exists, if not, insert it
		locationID, err := getLocationID(
			r, w, tx,
			&event.Location.Address, &event.Location.Stadium, // mandatory
			&event.Location.Capacity,
			&event.Location.Country,
		)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		// insert new event
		var eventID int
		eventQuery := `
				INSERT INTO Events (name, date, price, available_tickets, location_id)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
		`
		if err := tx.QueryRow(
			r.Context(), eventQuery,
			event.Name, rfc3339Date, event.Price, event.AvailableTickets,
			locationID,
		).Scan(&eventID); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to create the event.")
			return
		}

		if err = tx.Commit(r.Context()); err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to commit the transaction.",
			)
			return
		}

		writeJSONResponse(
			w,
			http.StatusCreated,
			models.SuccessResponseCreate{Message: "Event created successfully.", ID: eventID},
		)
	}
}

// UpdateEventHandler updates an existing event by ID.
//
//	@Summary			Update an existing event
//	@Description	Update event details based on the provided payload.
//	@ID						api.updateEvent
//	@Tags					events
//	@Produce			json
//	@Accept				json
//	@Param				id			path		string										true	"Event ID"
//	@Param				body		body		models.UpdateEventRequest	true	"Payload to update an event"
//	@Success			200		{object}	models.SuccessResponse		"Event updated successfully"
//	@Failure			400		{object}	models.ErrorResponse			"Bad Request"
//	@Failure			422		{object}	models.ErrorResponse			"Unprocessable Entity"
//	@Failure			500		{object}	models.ErrorResponse			"Internal Server Error"
//	@Router				/events/{id} [put]
func UpdateEventHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// parse the event ID from the URL
		vars := mux.Vars(r)
		eventID, ok := vars["id"]
		if !ok {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid event ID.")
			return
		}

		// parse the request body into UpdateEventRequest
		var eventPayload models.UpdateEventRequest
		if err := json.NewDecoder(r.Body).Decode(&eventPayload); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON input.")
			return
		}

		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction.")
			return
		}
		defer tx.Rollback(r.Context())

		var updateQueries []string
		var updateArgs []interface{}
		argIndex := 1

		// construct the update query
		if eventPayload.Name != nil {
			updateQueries = append(updateQueries, fmt.Sprintf("name = $%d", argIndex))
			updateArgs = append(updateArgs, *eventPayload.Name)
			argIndex++
		}
		if eventPayload.AvailableTickets != nil {
			updateQueries = append(updateQueries, fmt.Sprintf("available_tickets = $%d", argIndex))
			updateArgs = append(updateArgs, *eventPayload.AvailableTickets)
			argIndex++
		}
		if eventPayload.Date != nil {
			rfc3339Date, err := dateToRFC3339(w, *eventPayload.Date)
			if err != nil && rfc3339Date == "" {
				writeErrorResponse(
					w,
					http.StatusBadRequest,
					"Invalid date format; must be YYYY-MM-DD HH:MM or RFC3339.",
				)
				return
			}

			updateQueries = append(updateQueries, fmt.Sprintf("date = $%d", argIndex))
			updateArgs = append(updateArgs, rfc3339Date)
			argIndex++
		}
		if eventPayload.Price != nil {
			updateQueries = append(updateQueries, fmt.Sprintf("price = $%d", argIndex))
			updateArgs = append(updateArgs, *eventPayload.Price)
			argIndex++
		}
		if eventPayload.Location != nil {
			locationID, err := getLocationID(
				r, w, tx,
				eventPayload.Location.Address,
				eventPayload.Location.Stadium,
				eventPayload.Location.Capacity,
				eventPayload.Location.Country,
			)
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, err.Error())
				return
			}
			updateQueries = append(updateQueries, fmt.Sprintf("location_id = $%d", argIndex))
			updateArgs = append(updateArgs, locationID)
			argIndex++
		}

		if len(updateQueries) == 0 {
			writeErrorResponse(w, http.StatusUnprocessableEntity, "Nothing to update.")
			return
		}

		// append the args to the query
		updateArgs = append(updateArgs, eventID)
		updateQuery := fmt.Sprintf(
			"UPDATE Events SET %s WHERE id = $%d",
			strings.Join(updateQueries, ", "),
			argIndex,
		)

		_, err = tx.Exec(r.Context(), updateQuery, updateArgs...)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to update the event.")
			return
		}

		if err = tx.Commit(r.Context()); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to commit transaction.")
			return
		}

		writeJSONResponse(
			w,
			http.StatusOK,
			models.SuccessResponse{Message: "Event updated successfully."},
		)
	}
}

// DeleteEventHandler deletes an existing event by ID.
//
//	@Summary			Delete an existing event
//	@Description	Delete event by its ID.
//	@ID						api.deleteEvent
//	@Tags					events
//	@Produce			json
//	@Param				id		path			string												true	"Event ID"
//	@Success			200		{object}	models.SuccessResponseCreate	"Event deleted successfully"
//	@Failure			400		{object}	models.ErrorResponse					"Bad Request"
//	@Failure			500		{object}	models.ErrorResponse					"Internal Server Error"
//	@Router				/events/{id} [delete]
func DeleteEventHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		eventID, ok := vars["id"]
		if !ok {
			writeErrorResponse(w, http.StatusBadRequest, "Event ID not provided.")
			return
		}

		// delete the event
		query := `DELETE FROM Events WHERE id = $1`
		_, err := pool.Exec(r.Context(), query, eventID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete event.")
			return
		}

		writeJSONResponse(
			w,
			http.StatusOK,
			models.SuccessResponse{Message: "Event deleted successfully."},
		)
	}
}
