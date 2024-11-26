package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateReservationHandler handles the creation of reservations and associated tickets.
func CreateReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isRegistered(r) {
			handleError(w, http.StatusForbidden, "Forbidden: Insufficient permissions", nil)
			return
		}

		var reservationRequest struct {
			PrimaryUserID int `json:"primary_user_id"`
			EventID       int `json:"event_id"`
			TotalTickets  int `json:"total_tickets"`
			Tickets       []struct {
				Type string `json:"type"`
			} `json:"tickets"`
		}

		// parse the request body
		if err := json.NewDecoder(r.Body).Decode(&reservationRequest); err != nil {
			handleError(w, http.StatusBadRequest, "Invalid JSON input", err)
			return
		}

		// start database transaction
		tx, err := pool.Begin(r.Context())
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to start transaction", err)
			return
		}
		defer tx.Rollback(r.Context())

		// fetch the base price of the event
		price, err := getEventBasePrice(r, tx, reservationRequest.EventID)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch event base price", nil)
		}

		// fetch the available tickets
		availableTickets, err := getEventAvailableTickets(r, tx, reservationRequest.EventID)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Unable to fetch available tickets", nil)
		}

		if availableTickets < reservationRequest.TotalTickets {
			handleError(w, http.StatusBadRequest, "Not enough tickets available", nil)
		}

		reservationStatusID, err := fetchReservationStatus(r, tx, "CONFIRMED")
		if err != nil {
			handleError(
				w,
				http.StatusInternalServerError,
				"Failed to fetch reservation status",
				nil,
			)
		}

		// insert a reservation
		var reservationID string
		reservationQuery := `
            INSERT INTO Reservations (primary_user_id, event_id, total_tickets, status_id)
            VALUES ($1, $2, $3, $4)
            RETURNING id`
		err = tx.QueryRow(r.Context(), reservationQuery,
			reservationRequest.PrimaryUserID,
			reservationRequest.EventID,
			reservationRequest.TotalTickets,
			reservationStatusID,
		).Scan(&reservationID)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to create reservation", err)
			return
		}

		// create an appropriate amount of tickets
		ticketQuery := `
            INSERT INTO Tickets (event_id, reservation_id, price, type_id, status_id)
            VALUES ($1, $2, $3, $4, $5)`
		for _, ticket := range reservationRequest.Tickets {
			discount, err := fetchTicketDiscount(r, tx, ticket.Type)
			if err != nil {
				handleError(
					w,
					http.StatusInternalServerError,
					"Failed to fetch ticket discount",
					nil,
				)
			}

			typeID, err := fetchTicketType(r, tx, ticket.Type)
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to fetch ticket type", nil)
			}

			statusID, err := fetchTicketStatus(r, tx, "RESERVED")
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to fetch ticket status", nil)
			}

			_, err = tx.Exec(r.Context(), ticketQuery,
				reservationRequest.EventID,
				reservationID,
				price*(1-discount),
				typeID,
				statusID,
			)
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to create tickets", err)
				return
			}
		}

		// Commit transaction
		if err := tx.Commit(r.Context()); err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to commit transaction", err)
			return
		}

		// Respond with the reservation ID
		writeJSONResponse(w, http.StatusCreated, map[string]string{"reservation_id": reservationID})
	}
}

func GetReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// // Parse pagination parameters
		// query := r.URL.Query()
		// limit := 10
		// offset := 0
		// if qLimit := query.Get("limit"); qLimit != "" {
		// 	if l, err := strconv.Atoi(qLimit); err == nil && l > 0 {
		// 		limit = l
		// 	}
		// }
		// if qOffset := query.Get("offset"); qOffset != "" {
		// 	if o, err := strconv.Atoi(qOffset); err == nil && o >= 0 {
		// 		offset = o
		// 	}
		// }
		//
		// Fetch reservations
		reservationsQuery := `
			SELECT r.id, r.primary_user_id, r.event_id, r.total_tickets, s.name as status
			FROM Reservations r
			JOIN ReservationStatuses s ON r.status_id = s.id
			ORDER BY r.id`
		// LIMIT $1 OFFSET $2`
		rows, err := pool.Query(r.Context(), reservationsQuery) //, limit, offset)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch reservations", err)
			return
		}
		defer rows.Close()

		var reservations []struct {
			ID            string `json:"id"`
			PrimaryUserID int    `json:"primary_user_id"`
			EventID       int    `json:"event_id"`
			TotalTickets  int    `json:"total_tickets"`
			Status        string `json:"status"`
			Tickets       []struct {
				ID     string  `json:"id"`
				Type   string  `json:"type"`
				Price  float64 `json:"price"`
				Status string  `json:"status"`
			} `json:"tickets"`
		}

		// Iterate over each reservation
		for rows.Next() {
			var reservation struct {
				ID            string `json:"id"`
				PrimaryUserID int    `json:"primary_user_id"`
				EventID       int    `json:"event_id"`
				TotalTickets  int    `json:"total_tickets"`
				Status        string `json:"status"`
				Tickets       []struct {
					ID     string  `json:"id"`
					Type   string  `json:"type"`
					Price  float64 `json:"price"`
					Status string  `json:"status"`
				} `json:"tickets"`
			}

			if err := rows.Scan(
				&reservation.ID, &reservation.PrimaryUserID,
				&reservation.EventID, &reservation.TotalTickets, &reservation.Status,
			); err != nil {
				handleError(
					w,
					http.StatusInternalServerError,
					"Failed to parse reservation data",
					err,
				)
				return
			}

			// Fetch associated tickets
			ticketsQuery := `
				SELECT t.id, tt.name as type, t.price, ts.name as status
				FROM Tickets t
				JOIN TicketTypes tt ON t.type_id = tt.id
				JOIN TicketStatuses ts ON t.status_id = ts.id
				WHERE t.reservation_id = $1`
			ticketRows, err := pool.Query(r.Context(), ticketsQuery, reservation.ID)
			if err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to fetch tickets", err)
				return
			}
			defer ticketRows.Close()

			for ticketRows.Next() {
				var ticket struct {
					ID     string  `json:"id"`
					Type   string  `json:"type"`
					Price  float64 `json:"price"`
					Status string  `json:"status"`
				}
				if err := ticketRows.Scan(&ticket.ID, &ticket.Type, &ticket.Price, &ticket.Status); err != nil {
					handleError(
						w,
						http.StatusInternalServerError,
						"Failed to parse ticket data",
						err,
					)
					return
				}
				reservation.Tickets = append(reservation.Tickets, ticket)
			}

			reservations = append(reservations, reservation)
		}

		if rows.Err() != nil {
			handleError(
				w,
				http.StatusInternalServerError,
				"Error occurred while iterating reservations",
				rows.Err(),
			)
			return
		}

		// Respond with the list of reservations
		writeJSONResponse(w, http.StatusOK, reservations)
	}
}

func GetReservationByIDHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// parse the user id from the url
		vars := mux.Vars(r)
		id, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "User ID not provided in the query", nil)
			return
		}

		var reservation struct {
			ID            string `json:"id"`
			PrimaryUserID int    `json:"primary_user_id"`
			EventID       int    `json:"event_id"`
			TotalTickets  int    `json:"total_tickets"`
			Status        string `json:"status"`
			Tickets       []struct {
				ID     string  `json:"id"`
				Type   string  `json:"type"`
				Price  float64 `json:"price"`
				Status string  `json:"status"`
			} `json:"tickets"`
		}

		// Fetch reservation details
		reservationQuery := `
            SELECT r.id, r.primary_user_id, r.event_id, r.total_tickets, s.name as status
            FROM Reservations r
            JOIN ReservationStatuses s ON r.status_id = s.id
            WHERE r.id = $1`
		err := pool.QueryRow(r.Context(), reservationQuery, id).Scan(
			&reservation.ID, &reservation.PrimaryUserID, &reservation.EventID,
			&reservation.TotalTickets, &reservation.Status,
		)
		if err != nil {
			handleError(w, http.StatusNotFound, "Reservation not found", err)
			return
		}

		// Fetch associated tickets
		ticketsQuery := `
            SELECT t.id, tt.name as type, t.price, ts.name as status
            FROM Tickets t
            JOIN TicketTypes tt ON t.type_id = tt.id
            JOIN TicketStatuses ts ON t.status_id = ts.id
            WHERE t.reservation_id = $1`
		rows, err := pool.Query(r.Context(), ticketsQuery, id)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to fetch tickets", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var ticket struct {
				ID     string  `json:"id"`
				Type   string  `json:"type"`
				Price  float64 `json:"price"`
				Status string  `json:"status"`
			}
			if err := rows.Scan(&ticket.ID, &ticket.Type, &ticket.Price, &ticket.Status); err != nil {
				handleError(w, http.StatusInternalServerError, "Failed to parse ticket data", err)
				return
			}
			reservation.Tickets = append(reservation.Tickets, ticket)
		}

		writeJSONResponse(w, http.StatusOK, reservation)
	}
}

func UpdateReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// parse the user id from the url
		vars := mux.Vars(r)
		reservationId, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "User ID not provided in the query", nil)
			return
		}

		var updateRequest struct {
			Status string `json:"status"`
		}

		if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
			handleError(w, http.StatusBadRequest, "Invalid JSON input", err)
			return
		}

		statusID, err := fetchReservationStatusPool(r, pool, updateRequest.Status)
		if err != nil {
			handleError(w, http.StatusBadRequest, "Invalid status", err)
			return
		}

		query := `UPDATE Reservations SET status_id = $1 WHERE id = $2`
		_, err = pool.Exec(r.Context(), query, statusID, reservationId)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to update reservation", err)
			return
		}

		writeJSONResponse(
			w,
			http.StatusOK,
			map[string]string{"message": "Reservation updated successfully"},
		)
	}
}

func DeleteReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// parse the user id from the url
		vars := mux.Vars(r)
		id, ok := vars["id"]
		if !ok {
			handleError(w, http.StatusBadRequest, "User ID not provided in the query", nil)
			return
		}

		tx, err := pool.Begin(r.Context())
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to start transaction", err)
			return
		}
		defer tx.Rollback(r.Context())

		// Delete tickets associated with the reservation
		ticketsQuery := `DELETE FROM Tickets WHERE reservation_id = $1`
		_, err = tx.Exec(r.Context(), ticketsQuery, id)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to delete tickets", err)
			return
		}

		// Delete the reservation
		reservationQuery := `DELETE FROM Reservations WHERE id = $1`
		_, err = tx.Exec(r.Context(), reservationQuery, id)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to delete reservation", err)
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to commit transaction", err)
			return
		}

		writeJSONResponse(
			w,
			http.StatusOK,
			map[string]string{"message": "Reservation deleted successfully"},
		)
	}
}
