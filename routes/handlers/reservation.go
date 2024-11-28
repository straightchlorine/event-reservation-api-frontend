package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/models"
)

// Fetch all reservations with associated ticket details.
func GetReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// fetch all data associated with reservations
		query := `
			SELECT r.id, u.username, r.created_at, r.total_tickets, rs.name,
				e.name, e.date, l.country, l.address, l.stadium
			FROM Reservations r
			JOIN reservation_statuses rs ON r.status_id = rs.id
			JOIN Users u ON r.user_id = u.id
			JOIN Events e ON r.event_id = e.id
			JOIN Locations l ON e.location_id = l.id
		`
		rows, err := pool.Query(r.Context(), query)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch reservations", err)
			return
		}
		defer rows.Close()

		// build the response data
		reservations := []models.ReservationResponse{}
		for rows.Next() {
			var res models.ReservationResponse
			var location models.LocationResponse
			var event models.EventResponse

			// scan the rows
			err := rows.Scan(
				&res.ID,
				&res.Username,
				&res.CreatedAt,
				&res.TotalTickets,
				&res.Status,
				&event.Name,
				&event.Date,
				&location.Country,
				&location.Address,
				&location.Stadium,
			)
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to parse reservation", err)
				return
			}

			// append the data to the response
			event.Location = location
			res.Event = event

			// fetch tickets attributed to the reservation
			tickets, err := fetchTicketsForReservation(r.Context(), pool, res.ID)
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to fetch tickets", err)
				return
			}

			// append to the response
			res.Tickets = tickets
			reservations = append(reservations, res)
		}
		writeJSONResponse(w, http.StatusOK, reservations)
	}
}

// Create a reservation along with with associated tickets.
func CreateReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isRegistered(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		// get the user identifier of the logged in user
		userId, err := getUserIDFromContext(r.Context())
		if err != nil {
			handleError(
				w,
				http.StatusInternalServerError,
				"Failed to fetch the user identifier",
				nil,
			)
			return
		}

		// decode the request body
		var resPayload models.ReservationPayload
		if err := json.NewDecoder(r.Body).Decode(&resPayload); err != nil {
			handleError(w, http.StatusBadRequest, "Invalid JSON input", err)
			return
		}

		// validate the request
		if err := validateReservationRequest(resPayload); err != nil {
			handleError(w, http.StatusBadRequest, err.Error(), nil)
			return
		}

		// begin database transaction
		tx, err := pool.Begin(r.Context())
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to start transaction", err)
			return
		}
		defer tx.Rollback(r.Context())

		// fetch reservation details
		var req models.ReservationRequest
		basePrice, availableTickets, statusID, err := fetchReservationDetails(
			r,
			tx,
			"PENDING", // will be updated after inserting the tickets
			resPayload.EventID,
		)

		if err != nil {
			handleError(
				w,
				http.StatusInternalServerError,
				"Failed to fetch reservation details",
				err,
			)
			return
		}

		// assign already fetched values to the request struct
		req.UserID = userId
		req.EventID = resPayload.EventID
		req.TotalTickets = len(resPayload.Tickets)
		req.StatusID = statusID

		// check if there is enough tickets available
		if availableTickets < req.TotalTickets {
			handleError(w, http.StatusBadRequest, "Not enough tickets available", nil)
			return
		}

		substractTicketsFromEvent(r.Context(), tx, req.EventID, req.TotalTickets)

		// insert a reservation
		var reservationId string
		reservationQuery := `
			INSERT INTO Reservations (user_id, event_id, total_tickets, status_id)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		err = tx.QueryRow(r.Context(),
			reservationQuery,
			req.UserID,
			req.EventID,
			req.TotalTickets,
			req.StatusID,
		).Scan(&reservationId)

		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to create reservation", err)
			return
		}

		ticketQuery := `
			INSERT INTO Tickets (reservation_id, price, type_id, status_id)
			VALUES ($1, $2, $3, $4)
		`
		// insert the reserved tickets
		for _, ticket := range resPayload.Tickets {
			discount, typeId, statusId, err := fetchTicketDetails(
				r.Context(),
				tx,
				"RESERVED",
				ticket.Type,
			)
			if err != nil {
				handleError(
					w,
					http.StatusInternalServerError,
					"Failed to fetch ticket details",
					nil,
				)
				return
			}

			// execute the insert query
			_, err = tx.Exec(r.Context(), ticketQuery,
				reservationId,
				basePrice*(1-discount),
				typeId,
				statusId,
			)
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to create tickets", err)
				return
			}
		}

		err = confirmReservation(r.Context(), tx, reservationId)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to confirm reservation", err)
		}

		// commit the transaction
		if err := tx.Commit(r.Context()); err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to commit transaction", err)
			return
		}

		// respond with the reservation ID
		writeJSONResponse(
			w,
			http.StatusCreated,
			map[string]interface{}{
				"message":       "Reservation created successfully",
				"reservationId": reservationId,
			},
		)
	}
}

// Fetch a reservation by ID.
func GetReservationByIDHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reservationId := mux.Vars(r)["id"]

		var res models.ReservationResponse
		var location models.LocationResponse
		var event models.EventResponse

		// fetch the reservation details
		query := `
			SELECT r.id, u.username, r.created_at, r.total_tickets, rs.name,
				e.name, e.date, l.country, l.address, l.stadium
			FROM Reservations r
			JOIN reservation_statuses rs ON r.status_id = rs.id
			JOIN Users u ON r.user_id = u.id
			JOIN Events e ON r.event_id = e.id
			JOIN Locations l ON e.location_id = l.id
			WHERE r.id = $1
		`
		if err := pool.QueryRow(r.Context(), query, reservationId).Scan(
			&res.ID,
			&res.Username,
			&res.CreatedAt,
			&res.TotalTickets,
			&res.Status,
			&event.Name,
			&event.Date,
			&location.Country,
			&location.Address,
			&location.Stadium,
		); err != nil {
			handleError(w, http.StatusNotFound, "Reservation not found", err)
			return
		}

		// fetch the tickets associated with the reservation
		tickets, err := fetchTicketsForReservation(r.Context(), pool, res.ID)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch tickets", err)
			return
		}

		// append to the response
		event.Location = location
		res.Tickets = tickets
		res.Event = event

		writeJSONResponse(w, http.StatusOK, res)
	}
}

// TODO: Maybe implement also update handler, for now skipped.

// Delete a reservation and its associated tickets.
func DeleteReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reservationId := mux.Vars(r)["id"]

		tx, err := pool.Begin(r.Context())
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to start transaction", err)
			return
		}
		defer tx.Rollback(r.Context())

		// Verify if the reservation exists
		var exists bool
		checkReservationQuery := `SELECT EXISTS(SELECT 1 FROM Reservations WHERE id = $1)`
		if err := tx.QueryRow(r.Context(), checkReservationQuery, reservationId).Scan(&exists); err != nil {
			handleError(
				w,
				http.StatusInternalServerError,
				"Failed to verify reservation existence",
				err,
			)
			return
		}
		if !exists {
			handleError(w, http.StatusNotFound, "Reservation not found", nil)
			return
		}

		deleteTicketsQuery := `DELETE FROM Tickets WHERE reservation_id = $1`
		if _, err := tx.Exec(r.Context(), deleteTicketsQuery, reservationId); err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to delete tickets", err)
			return
		}

		deleteReservationQuery := `DELETE FROM Reservations WHERE id = $1`
		if _, err := tx.Exec(r.Context(), deleteReservationQuery, reservationId); err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to delete reservation", err)
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to commit transaction", err)
			return
		}

		writeJSONResponse(w, http.StatusOK,
			map[string]string{
				"message": "Reservation deleted successfully",
			})
	}
}
