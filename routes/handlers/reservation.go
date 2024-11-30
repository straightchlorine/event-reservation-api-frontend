package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/models"
)

// GetReservationHandler lists all reservations.
//
//	@Summary		List all reservations (admin only).
//	@Description	Retrieve a list of all reservations, including their details and tickets they reserve.
//	@Tags			reservations
//	@ID				api.getReservations
//	@Produce		json
//	@Success		200	{object}	models.ReservationsResponse	"List of reservations"
//	@Failure		403	{object}	models.ErrorResponse		"Forbidden"
//	@Failure		404	{object}	models.ErrorResponse		"Not Found"
//	@Failure		500	{object}	models.ErrorResponse		"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/reservations [get]
func GetReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			writeErrorResponse(w, http.StatusBadRequest, "Insufficient permissions.")
			return
		}

		query := `
			SELECT r.id, u.username, r.created_at, r.total_tickets, rs.name,
				e.id, e.name, e.date, l.country, l.address, l.stadium
			FROM Reservations r
			JOIN reservation_statuses rs ON r.status_id = rs.id
			JOIN Users u ON r.user_id = u.id
			JOIN Events e ON r.event_id = e.id
			JOIN Locations l ON e.location_id = l.id
		`
		rows, err := pool.Query(r.Context(), query)
		if err != nil {
			if err == pgx.ErrNoRows {
				writeErrorResponse(w, http.StatusNotFound, "Reservations not found.")
				return
			}
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch reservations.",
			)
			return
		}
		defer rows.Close()

		// build the response data
		reservations := []models.ReservationResponse{}
		for rows.Next() {
			var res models.ReservationResponse
			var location models.LocationResponse
			var event models.EventResponse

			if err := rows.Scan(
				&res.ID, &res.Username, &res.CreatedAt, &res.TotalTickets, &res.Status,
				&event.ID, &event.Name, &event.Date,
				&location.Country, &location.Address, &location.Stadium,
			); err != nil {
				writeErrorResponse(
					w,
					http.StatusInternalServerError,
					"Failed to parse reservation.",
				)
				return
			}

			// append the structs to the response
			event.Location = location
			res.Event = event

			tickets, err := fetchTickets(r.Context(), pool, res.ID)
			if err != nil {
				writeErrorResponse(
					w,
					http.StatusInternalServerError,
					"Failed to fetch the tickets.",
				)
				return
			}

			// append ticket struct to the response
			res.Tickets = tickets
			reservations = append(reservations, res)
		}
		reservations_response := models.ReservationsResponse{Reservations: reservations}
		writeJSONResponse(w, http.StatusOK, reservations_response)
	}
}

// GetReservationByIDHandler returns a handler function that returns a single reservation.
//
//	@Summary		Get a reservation by ID (admin/owner only).
//	@Description	Retrieve a single reservation, including their details and tickets they reserve.
//	@Tags			reservations
//	@ID				api.getReservationsByID
//	@Produce		json
//	@Param			id	path		string						true	"Reservation ID"
//	@Success		200	{object}	models.ReservationResponse	"Reservation details"
//	@Failure		403	{object}	models.ErrorResponse		"Forbidden"
//	@Failure		404	{object}	models.ErrorResponse		"Not Found"
//	@Failure		500	{object}	models.ErrorResponse		"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/reservations/{id} [get]
func GetReservationByIDHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the user id
		userId, err := getUserIdFromContext(r.Context())
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch the user identifier.",
			)
			return
		}

		// permissions
		if !isAdmin(r) && !isOwner(r, userId) {
			writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions.")
			return
		}

		reservationId, err := parseReservationIdFromURL(r)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err.Error())
		}

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
			writeErrorResponse(w, http.StatusNotFound, "Reservation not found.")
			return
		}

		// fetch the tickets associated with the reservation
		tickets, err := fetchTickets(r.Context(), pool, res.ID)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch tickets")
			return
		}

		// append to the response
		event.Location = location
		res.Tickets = tickets
		res.Event = event

		writeJSONResponse(w, http.StatusOK, res)
	}
}

// GetCurrentUserReservationsHandler lists all reservations for currently logged in user.
//
//	@Summary		List user reservations for currently logged in user.
//	@Description	Retrieve a list of current user's reservations along with details and tickets they reserve.
//	@Tags			reservations
//	@ID				api.getReservationsForCurrentUser
//	@Produce		json
//	@Success		200	{object}	models.ReservationsResponse	"List of reservations for the user"
//	@Failure		400	{object}	models.ErrorResponse		"Bad Request"
//	@Failure		404	{object}	models.ErrorResponse		"Not Found"
//	@Failure		500	{object}	models.ErrorResponse		"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/reservations/user [get]
func GetCurrentUserReservationsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the user ID out of context
		userID, err := getUserIdFromContext(r.Context())
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch the user id.",
			)
			return
		}

		if !isOverUnregistered(r) {
			writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions.")
			return
		}

		query := `
			SELECT r.id, u.username, r.created_at, r.total_tickets, rs.name,
				e.id, e.name, e.date, l.country, l.address, l.stadium
			FROM Reservations r
			JOIN reservation_statuses rs ON r.status_id = rs.id
			JOIN Users u ON r.user_id = u.id
			JOIN Events e ON r.event_id = e.id
			JOIN Locations l ON e.location_id = l.id
			WHERE r.user_id = $1
		`
		rows, err := pool.Query(r.Context(), query, userID)
		if err != nil {
			if err == pgx.ErrNoRows {
				writeErrorResponse(w, http.StatusNotFound, "No reservations found for the user.")
			}
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch reservations.",
			)
			return
		}
		defer rows.Close()

		// build the response data
		reservations := []models.ReservationResponse{}
		for rows.Next() {
			var res models.ReservationResponse
			var location models.LocationResponse
			var event models.EventResponse

			if err := rows.Scan(
				&res.ID, &res.Username, &res.CreatedAt, &res.TotalTickets, &res.Status,
				&event.ID, &event.Name, &event.Date,
				&location.Country, &location.Address, &location.Stadium,
			); err != nil {
				if err == pgx.ErrNoRows {
					writeErrorResponse(
						w,
						http.StatusNotFound,
						"No reservations found for the user.",
					)
					return
				}
				writeErrorResponse(
					w,
					http.StatusInternalServerError,
					"Failed to parse reservation.",
				)
				return
			}

			// append the structs to the response
			event.Location = location
			res.Event = event

			tickets, err := fetchTickets(r.Context(), pool, res.ID)
			if err != nil {
				writeErrorResponse(
					w,
					http.StatusInternalServerError,
					"Failed to fetch the tickets.",
				)
				return
			}

			// append ticket struct to the response
			res.Tickets = tickets
			reservations = append(reservations, res)
		}

		if len(reservations) == 0 {
			writeErrorResponse(w, http.StatusNotFound, "No reservations found for the user.")
			return
		}

		writeJSONResponse(w, http.StatusOK, reservations)
	}
}

// GetCurrentUserReservationsTicketsHandler lists all tickets for currently logged in user.
//
//	@Summary		List user tickets for currently logged in user.
//	@Description	Retrieve a list of current user's tickets.
//	@Tags			reservations
//	@ID				api.getReservationTicketsForCurrentUser
//	@Produce		json
//	@Param			id	path		string						true	"User ID"
//	@Success		200	{object}	models.UserTicketsResponse	"List of tickets belonging to the user"
//	@Failure		400	{object}	models.ErrorResponse		"Bad Request"
//	@Failure		404	{object}	models.ErrorResponse		"Not Found"
//	@Failure		500	{object}	models.ErrorResponse		"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/reservations/user/tickets [get]
func GetCurrentUserReservationsTicketsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the user ID out of context
		userID, err := getUserIdFromContext(r.Context())
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch the user id.",
			)
			return
		}

		if !isOverUnregistered(r) {
			writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions.")
			return
		}

		// fetch all tickets user has bought
		query := `
			SELECT
				t.id, t.reservation_id, t.price,
				tt.name, ts.name,
				e.id, e.name, e.date,
				l.country, l.address, l.stadium
			FROM tickets t
			JOIN ticket_types tt ON t.type_id = tt.id
			JOIN ticket_statuses ts ON t.status_id = ts.id
			JOIN reservations r ON t.reservation_id = r.id
			JOIN events e ON r.event_id = e.id
			JOIN locations l ON e.location_id = l.id
			WHERE r.user_id = $1
		`
		rows, err := pool.Query(r.Context(), query, userID)
		if err != nil {
			if err == pgx.ErrNoRows {
				writeErrorResponse(w, http.StatusNotFound, "No tickets found for the user.")
			}
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch tickets.",
			)
			return
		}
		defer rows.Close()

		// build the response data
		tickets := []models.UserTicketResponse{}
		for rows.Next() {
			var ticket models.UserTicketResponse
			var location models.LocationResponse
			var event models.EventResponse

			if err := rows.Scan(
				&ticket.ID, &ticket.ReservationID, &ticket.Price, &ticket.Type, &ticket.Status,
				&event.ID, &event.Name, &event.Date,
				&location.Country, &location.Address, &location.Stadium,
			); err != nil {
				writeErrorResponse(
					w,
					http.StatusInternalServerError,
					"Failed to parse the tickets.",
				)
				return
			}

			// append the structs to the response
			event.Location = location
			ticket.Event = event

			tickets = append(tickets, ticket)
		}

		if len(tickets) == 0 {
			writeErrorResponse(w, http.StatusNotFound, "No tickets found for the user.")
			return
		}

		tickets_respone := models.UserTicketsResponse{UserID: userID, Tickets: tickets}
		writeJSONResponse(w, http.StatusOK, tickets_respone)
	}
}

// GetUserReservationsHandler returns all reservations for a specific user.
//
//	@Summary		List user reservations (admin/owner only)
//	@Description	Retrieve a list of all reservations made by a specific user, including their details and tickets they reserve.
//	@Tags			reservations
//	@ID				api.getReservationsForUserByID
//	@Produce		json
//	@Param			user_id	query		int							true	"User ID"
//	@Success		200		{object}	models.ReservationsResponse	"List of reservations for the user"
//	@Failure		400		{object}	models.ErrorResponse		"Bad Request"
//	@Failure		404		{object}	models.ErrorResponse		"Not Found"
//	@Failure		500		{object}	models.ErrorResponse		"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/reservations/user/{id} [get]
func GetUserReservationsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the user id
		userId, err := getUserIdFromContext(r.Context())
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch the user identifier.",
			)
			return
		}

		// permissions
		if !isAdmin(r) && !isOwner(r, userId) {
			writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions.")
			return
		}

		query := `
			SELECT r.id, u.username, r.created_at, r.total_tickets, rs.name,
				e.id, e.name, e.date, l.country, l.address, l.stadium
			FROM Reservations r
			JOIN reservation_statuses rs ON r.status_id = rs.id
			JOIN Users u ON r.user_id = u.id
			JOIN Events e ON r.event_id = e.id
			JOIN Locations l ON e.location_id = l.id
			WHERE r.user_id = $1
		`
		rows, err := pool.Query(r.Context(), query, userId)
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch reservations.",
			)
			return
		}
		defer rows.Close()

		// build the response data
		reservations := []models.ReservationResponse{}
		for rows.Next() {
			var res models.ReservationResponse
			var location models.LocationResponse
			var event models.EventResponse

			if err := rows.Scan(
				&res.ID, &res.Username, &res.CreatedAt, &res.TotalTickets, &res.Status,
				&event.ID, &event.Name, &event.Date,
				&location.Country, &location.Address, &location.Stadium,
			); err != nil {
				if err == pgx.ErrNoRows {
					writeErrorResponse(
						w,
						http.StatusNotFound,
						"No reservations found for the user.",
					)
					return
				}
				writeErrorResponse(
					w,
					http.StatusInternalServerError,
					"Failed to parse reservation.",
				)
				return
			}

			// append the structs to the response
			event.Location = location
			res.Event = event

			tickets, err := fetchTickets(r.Context(), pool, res.ID)
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch the tickets")
				return
			}

			// append ticket struct to the response
			res.Tickets = tickets
			reservations = append(reservations, res)
		}

		if len(reservations) == 0 {
			writeErrorResponse(w, http.StatusNotFound, "No reservations found for the user.")
			return
		}

		reservations_response := models.ReservationsResponse{Reservations: reservations}
		writeJSONResponse(w, http.StatusOK, reservations_response)
	}
}

// GetUserReservationsTicketsHandler returns all tickets for a specific user.
//
//	@Summary		List user tickets (admin/owner only).
//	@Description	Retrieve a list of user's tickets.
//	@Tags			reservations
//	@ID				api.getReservationTicketsForUserByID
//	@Produce		json
//	@Param			id	path		string						true	"User ID"
//	@Success		200	{object}	models.UserTicketsResponse	"List of tickets belonging to the user"
//	@Failure		400	{object}	models.ErrorResponse		"Bad Request"
//	@Failure		404	{object}	models.ErrorResponse		"Not Found"
//	@Failure		500	{object}	models.ErrorResponse		"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/reservations/user/{id}/tickets [get]
func GetUserReservationsTicketsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the user id
		userID, err := getUserIdFromContext(r.Context())
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch the user identifier.",
			)
			return
		}

		// permissions
		if !isAdmin(r) && !isOwner(r, userID) {
			writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions.")
			return
		}

		// fetch all tickets user has bought
		query := `
			SELECT
				t.id, t.reservation_id, t.price,
				tt.name, ts.name,
				e.id, e.name, e.date,
				l.country, l.address, l.stadium
			FROM tickets t
			JOIN ticket_types tt ON t.type_id = tt.id
			JOIN ticket_statuses ts ON t.status_id = ts.id
			JOIN reservations r ON t.reservation_id = r.id
			JOIN events e ON r.event_id = e.id
			JOIN locations l ON e.location_id = l.id
			WHERE r.user_id = $1
		`
		rows, err := pool.Query(r.Context(), query, userID)
		if err != nil {
			if err == pgx.ErrNoRows {
				writeErrorResponse(w, http.StatusNotFound, "No tickets found for the user.")
			}
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch tickets.",
			)
			return
		}
		defer rows.Close()

		// build the response data
		tickets := []models.UserTicketResponse{}
		for rows.Next() {
			var ticket models.UserTicketResponse
			var location models.LocationResponse
			var event models.EventResponse

			if err := rows.Scan(
				&ticket.ID, &ticket.ReservationID, &ticket.Price, &ticket.Type, &ticket.Status,
				&event.ID, &event.Name, &event.Date,
				&location.Country, &location.Address, &location.Stadium,
			); err != nil {
				writeErrorResponse(
					w,
					http.StatusInternalServerError,
					"Failed to parse the tickets.",
				)
				return
			}

			// append the structs to the response
			event.Location = location
			ticket.Event = event

			tickets = append(tickets, ticket)
		}

		if len(tickets) == 0 {
			writeErrorResponse(w, http.StatusNotFound, "No tickets found for the user.")
			return
		}

		tickets_respone := models.UserTicketsResponse{UserID: userID, Tickets: tickets}
		writeJSONResponse(w, http.StatusOK, tickets_respone)
	}
}

// GetReservationTicketsHandler lists all tickets for a specific reservation.
//
//	@Summary		List tickets attributed to given reservation (owner/admin only).
//	@Description	Retrieve all tickets associated with a specific reservation by its ID.
//	@Tags			reservations
//	@ID				api.getReservationTicketsByID
//	@Produce		json
//	@Param			id	path		int									true	"Reservation ID"
//	@Success		200	{object}	models.ReservationTicketsResponse	"List of tickets for the reservation"
//	@Failure		400	{object}	models.ErrorResponse				"Bad Request"
//	@Failure		404	{object}	models.ErrorResponse				"Not Found"
//	@Failure		500	{object}	models.ErrorResponse				"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/reservations/{id}/tickets [get]
func GetReservationTicketsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reservationId, err := parseReservationIdFromURL(r)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err.Error())
		}

		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to start transaction.",
			)
			return
		}
		defer tx.Rollback(r.Context())

		var userId string
		if err := tx.QueryRow(
			r.Context(),
			"SELECT user_id FROM Reservations WHERE id = $1",
			reservationId,
		).Scan(&userId); err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch reservation user.",
			)
			return
		}

		// only available for admins and owners
		if !isAdmin(r) && !isOwner(r, userId) {
			writeErrorResponse(
				w,
				http.StatusBadRequest,
				"Insufficient permissions to get tickets of given reservation user.",
			)
			return
		}

		// query to fetch tickets for the given reservation ID
		query := `
			SELECT t.id, t.price, tt.name, ts.name
			FROM tickets t
			JOIN reservations r ON t.reservation_id = r.id
			JOIN ticket_types tt ON t.type_id = tt.id
			JOIN ticket_statuses ts ON t.status_id = ts.id
			WHERE r.id = $1
		`

		rows, err := tx.Query(r.Context(), query, reservationId)
		if err != nil {
			if err == pgx.ErrNoRows {
				writeErrorResponse(
					w,
					http.StatusNotFound,
					"No tickets found for this reservation.",
				)
				return
			}
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch tickets.")
			return
		}
		defer rows.Close()

		// build the response data
		tickets := []models.TicketResponse{}
		for rows.Next() {
			var ticket models.TicketResponse
			if err := rows.Scan(&ticket.ID, &ticket.Price, &ticket.Type, &ticket.Status); err != nil {
				writeErrorResponse(
					w,
					http.StatusInternalServerError,
					"Failed to parse ticket data.",
				)
				return
			}
			tickets = append(tickets, ticket)
		}

		// commit the transaction
		if err := tx.Commit(r.Context()); err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to commit transaction.",
			)
			return
		}

		// if no tickets found, return 404
		if len(tickets) == 0 {
			writeErrorResponse(w, http.StatusNotFound, "No tickets found for this reservation.")
			return
		}

		// reseponse with ids attached
		reservation_tickets := models.ReservationTicketsResponse{
			ReservationID: reservationId,
			UserID:        userId,
			Tickets:       tickets,
		}
		writeJSONResponse(w, http.StatusOK, reservation_tickets)
	}
}

// CreateReservationHandler creates a single reservation in the database along with its tickets.
//
//	@Summary		Create a reservation (owner/admin only).
//	@Description	Parse provided payload and create reservation and tickets within the database.
//	@Tags			reservations
//	@ID				api.createReservation
//	@Produce		json
//	@Param			body	body		models.CreateReservationPayload		true	"Payload to create a reservation"
//	@Success		200		{object}	models.SuccessResponseCreateUUID	"Reservation created successfully"
//	@Failure		403		{object}	models.ErrorResponse				"Forbidden"
//	@Failure		404		{object}	models.ErrorResponse				"Not Found"
//	@Failure		500		{object}	models.ErrorResponse				"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/reservations [put]
func CreateReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isOverUnregistered(r) {
			writeErrorResponse(
				w,
				http.StatusForbidden,
				"Insufficient permissions to create a reservation.",
			)
			return
		}

		// get the user identifier of the logged in user
		userId, err := getUserIdFromContext(r.Context())
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch the user identifier.",
			)
			return
		}

		// decode the request body
		var resPayload models.CreateReservationPayload
		if err := json.NewDecoder(r.Body).Decode(&resPayload); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON input")
			return
		}

		// validate the request
		if err := validateReservationRequest(resPayload); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// ensure atomicity during the process
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to start transaction.",
			)
			return
		}
		defer tx.Rollback(r.Context())

		// fetch reservation details, initial status will be pending
		// after creating tickets, will change to confirmed
		var req models.ReservationRequest
		basePrice, availableTickets, statusID, err := fetchReservationDetails(
			r, tx, "PENDING", resPayload.EventID,
		)
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch reservation details.",
			)
			return
		}

		// assign fetched values to the request struct
		req.UserID = userId
		req.EventID = resPayload.EventID
		req.TotalTickets = len(resPayload.Tickets)
		req.StatusID = statusID

		// check if there is enough tickets available
		if availableTickets < req.TotalTickets {
			writeErrorResponse(
				w,
				http.StatusBadRequest,
				"Not enough tickets to create a reservation.",
			)
			return
		}

		err = setAvailableTickets(r.Context(), tx, req.EventID, req.TotalTickets)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		}

		// insert a reservation
		var reservationId string
		reservationQuery := `
			INSERT INTO Reservations (user_id, event_id, total_tickets, status_id)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		if err = tx.QueryRow(r.Context(),
			reservationQuery,
			req.UserID,
			req.EventID,
			req.TotalTickets,
			req.StatusID,
		).Scan(&reservationId); err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to create a reservation.",
			)
			return
		}

		// insert the reserved tickets
		ticketQuery := `
			INSERT INTO Tickets (reservation_id, price, type_id, status_id)
			VALUES ($1, $2, $3, $4)
		`
		for _, ticket := range resPayload.Tickets {
			// initial state for tickets is RESERVED, later turns to SOLD
			discount, typeId, statusId, err := fetchTicketDetails(
				r.Context(), tx, "RESERVED", ticket.Type,
			)
			if err != nil {
				writeErrorResponse(
					w,
					http.StatusInternalServerError,
					"Failed to fetch ticket details.",
				)
				return
			}

			// execute the insert query
			if _, err = tx.Exec(
				r.Context(),
				ticketQuery,
				reservationId,
				basePrice*(1-discount),
				typeId,
				statusId,
			); err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, "Failed to create tickets.")
				return
			}
		}

		// confirms the reservations and 'sells' the tickets
		err = confirmReservation(r.Context(), tx, reservationId)
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to confirm reservation.",
			)
		}

		// commit the transaction
		if err := tx.Commit(r.Context()); err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to commit transaction.",
			)
			return
		}

		// respond with the reservation ID
		writeJSONResponse(
			w,
			http.StatusCreated,
			models.SuccessResponseCreateUUID{
				Message: "Reservation created successfully.",
				UUID:    reservationId,
			},
		)
	}
}

// CancelReservationHandler updates the status of the reservation and its tickets to CANCELLED.
//
//	@Summary		Cancel a reservation (owner/admin only).
//	@Description	Set statuses of reservation and its tickets to cancelled.
//	@Tags			reservations
//	@ID				api.cancelReservation
//	@Produce		json
//	@Param			id	path		string					true	"Reservation ID"
//	@Success		200	{object}	models.SuccessResponse	"Reservation canceled successfully"
//	@Failure		403	{object}	models.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	models.ErrorResponse	"Not Found"
//	@Failure		500	{object}	models.ErrorResponse	"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/reservations/{id}/cancel [post]
func CancelReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the user id
		userId, err := getUserIdFromContext(r.Context())
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to fetch the user identifier.",
			)
			return
		}

		// permissions
		if !isAdmin(r) && !isOwner(r, userId) {
			writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions.")
			return
		}

		reservationId, err := parseReservationIdFromURL(r)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err.Error())
		}

		// start a transaction
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to start transaction.")
			return
		}
		defer tx.Rollback(r.Context())

		if err := updateTicketsStatus(r.Context(), tx, reservationId, "CANCELLED"); err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to cancel the tickets.",
			)
			return
		}

		if err := updateReservationStatus(r.Context(), tx, reservationId, "CANCELLED"); err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to cancel the reservation.",
			)
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
			http.StatusOK,
			models.SuccessResponse{Message: "Reservation canceled successfully."},
		)
	}
}

// DeleteReservationHandler deletes a single reservation along with its tickets.
//
//	@Summary		Delete a reservation by ID (admin only).
//	@Description	Delete a single reservation along with its tickets from the database.
//	@Tags			reservations
//	@ID				api.deleteReservation
//	@Produce		json
//	@Param			id	path		string					true	"Reservation ID"
//	@Success		200	{object}	models.SuccessResponse	"Reservation deleted successfully"
//	@Failure		403	{object}	models.ErrorResponse	"Forbidden"
//	@Failure		404	{object}	models.ErrorResponse	"Not Found"
//	@Failure		500	{object}	models.ErrorResponse	"Internal Server Error"
//	@Security		BearerAuth
//	@Router			/reservations/{id} [delete]
func DeleteReservationHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			writeErrorResponse(
				w,
				http.StatusBadRequest,
				"Insufficient permissions to delete selected reservation.",
			)
			return
		}

		reservationId, err := parseReservationIdFromURL(r)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err.Error())
		}

		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to start a transaction.",
			)
			return
		}
		defer tx.Rollback(r.Context())

		// check if the reservation exists
		var exists bool
		checkReservationQuery := `SELECT EXISTS(SELECT 1 FROM Reservations WHERE id = $1)`
		if err := tx.QueryRow(r.Context(), checkReservationQuery, reservationId).Scan(&exists); err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to verify reservation existence.",
			)
			return
		}
		if !exists {
			writeErrorResponse(w, http.StatusNotFound, "Reservation not found.")
			return
		}

		deleteTicketsQuery := `DELETE FROM Tickets WHERE reservation_id = $1`
		if _, err := tx.Exec(r.Context(), deleteTicketsQuery, reservationId); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete tickets.")
			return
		}

		deleteReservationQuery := `DELETE FROM Reservations WHERE id = $1`
		if _, err := tx.Exec(r.Context(), deleteReservationQuery, reservationId); err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to delete reservation.",
			)
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to commit transaction.",
			)
			return
		}

		writeJSONResponse(
			w,
			http.StatusOK,
			models.SuccessResponse{Message: "Reservation deleted successfully."},
		)
	}
}
