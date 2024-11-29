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
	"event-reservation-api/models"
)

func confirmReservation(
	ctx context.Context,
	tx pgx.Tx,
	reservationId string) error {

	err := updateReservationStatus(ctx, tx, reservationId, "CONFIRMED")
	if err != nil {
		return fmt.Errorf("Failed to update reservation status: %w", err)
	}

	err = updateTicketsStatus(ctx, tx, reservationId, "SOLD")
	if err != nil {
		return fmt.Errorf("Failed to update ticket status: %w", err)
	}
	return nil
}

// Update the status of a reservation.
func updateTicketsStatus(
	ctx context.Context,
	tx pgx.Tx,
	resId string,
	status string) error {

	query := `
		UPDATE Tickets
		SET status_id = (
			SELECT id
			FROM ticket_statuses
			WHERE name = $1
			LIMIT 1
		)
		WHERE reservation_id = $2
	`
	_, err := tx.Exec(ctx, query, status, resId)
	if err != nil {
		return fmt.Errorf("failed to update reservation status: %w", err)
	}
	return nil
}

// Update the status of a reservation.
func updateReservationStatus(
	ctx context.Context,
	tx pgx.Tx,
	resId string,
	status string) error {

	query := `
		UPDATE reservations
		SET status_id = (
			SELECT id
			FROM reservation_statuses
			WHERE name = $1
			LIMIT 1
		)
		WHERE id = $2
	`
	_, err := tx.Exec(ctx, query, status, resId)
	if err != nil {
		return fmt.Errorf("failed to update reservation status: %w", err)
	}
	return nil
}

func substractTicketsFromEvent(
	ctx context.Context,
	tx pgx.Tx,
	eventID int,
	tickets int) error {

	query := `
		UPDATE events
		SET available_tickets = $2
		WHERE id = $1
	`
	_, err := tx.Exec(ctx, query, eventID, tickets)
	if err != nil {
		return fmt.Errorf("Failed to update available tickets for event %d: %w", eventID, err)
	}

	return nil
}

func fetchTicketDetails(
	ctx context.Context,
	tx pgx.Tx,
	ticketStatus string,
	ticketType string) (float64, int, int, error) {
	query := `
		SELECT
			tt.discount,
			tt.id,
			ts.id
		FROM ticket_types tt
		JOIN ticket_statuses ts ON ts.name = $2
		WHERE tt.name = $1
	`
	var discount float64
	var typeId int
	var statusId int

	err := tx.QueryRow(ctx, query, ticketType, ticketStatus).
		Scan(&discount, &typeId, &statusId)
	if err != nil {
		return 0.0, 0, 0, fmt.Errorf("Failed to fetch reservation details: %w", err)
	}

	return discount, typeId, statusId, nil
}

// Fetch the details required for creating a reservation.
//
// This involves base price, available tickets as well as the id of the
// reservation status
func fetchReservationDetails(
	r *http.Request,
	tx pgx.Tx,
	status string,
	eventID int) (float64, int, int, error) {
	query := `
		SELECT
			e.price,
			e.available_tickets,
			rs.id
		FROM events e
		JOIN reservation_statuses rs ON rs.name = $2
		WHERE e.id = $1
	`
	var basePrice float64
	var availableTickets int
	var statusID int

	err := tx.QueryRow(r.Context(), query, eventID, status).
		Scan(&basePrice, &availableTickets, &statusID)
	if err != nil {
		return 0.0, 0, 0, fmt.Errorf("Failed to fetch reservation details: %w", err)
	}

	return basePrice, availableTickets, statusID, nil
}

// Validate the reservation request payload.
func validateReservationRequest(req models.ReservationPayload) error {
	if req.EventID <= 0 {
		return fmt.Errorf("Invalid input: ensure all fields are non-negative")
	}
	return nil
}

// Fetch tickets for a reservation
func fetchTicketsForReservation(
	ctx context.Context,
	pool *pgxpool.Pool,
	reservationID string,
) ([]models.TicketResponse, error) {
	query := `
		SELECT t.id, t.price, ts.name AS status, tt.name AS type
		FROM Tickets t
		JOIN ticket_statuses ts ON t.status_id = ts.id
		JOIN ticket_types tt ON t.type_id = tt.id
		WHERE t.reservation_id = $1
	`

	rows, err := pool.Query(ctx, query, reservationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// build the list of tickets attributed to the reservation
	tickets := []models.TicketResponse{}
	for rows.Next() {
		var ticket models.TicketResponse
		err := rows.Scan(&ticket.ID, &ticket.Price, &ticket.Status, &ticket.Type)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

// Build a WHERE clause for filtering locations.
// If both provided: WHERE address = $1 AND stadium = $2; otherwise OR is used.
func getWhereClause(address *string, stadium *string) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if address != nil && *address != "" {
		conditions = append(conditions, "address = $"+fmt.Sprint(len(args)+1))
		args = append(args, *address)
	}
	if stadium != nil && *stadium != "" {
		conditions = append(conditions, "stadium = $"+fmt.Sprint(len(args)+1))
		args = append(args, *stadium)
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

func validateAddressAndStadium(address *string, stadium *string) error {
	if (address == nil || *address == "") && (stadium == nil || *stadium == "") {
		return fmt.Errorf("insufficient location data: address or stadium must be provided")
	}
	return nil
}

func insertLocation(
	ctx context.Context,
	tx pgx.Tx,
	address *string,
	stadium *string,
	capacity *int,
	country *string,
) (int, error) {
	var locationID int
	query := `
		INSERT INTO Locations (address, stadium, capacity, country)
		VALUES ($1, $2, COALESCE($3, 0), COALESCE($4, ''))
		RETURNING id
	`
	err := tx.QueryRow(
		ctx, query,
		address, stadium, capacity, country,
	).Scan(&locationID)

	if err != nil {
		return -1, fmt.Errorf("failed to insert a new location: %w", err)
	}
	return locationID, nil
}

// Check if a location exists within the database, if it does not, insert it.
func getLocationID(
	r *http.Request, w http.ResponseWriter, tx pgx.Tx,
	address *string,
	stadium *string,
	capacity *int,
	country *string,
) (int, error) {

	// validate the location data
	if err := validateAddressAndStadium(address, stadium); err != nil {
		return -1, err
	}

	// build appropriate WHERE clause
	whereClause, args := getWhereClause(address, stadium)
	if whereClause == "" {
		return -1, fmt.Errorf("Failed to build a valid query.")
	}

	// execute the query
	query := `
		SELECT id
		FROM Locations
	` + whereClause
	var locationID int
	err := tx.QueryRow(r.Context(), query, args...).Scan(&locationID)

	// if the location does not exist, insert it, otherwise return the ID
	if err != nil {
		if err == pgx.ErrNoRows {
			return insertLocation(r.Context(), tx, address, stadium, capacity, country)
		}
		return -1, fmt.Errorf("Failed to check existing location.")
	}

	return locationID, nil
}

// Utilitity to convert a date string to RFC3339 format.
func dateToRFC3339(w http.ResponseWriter, date string) (string, error) {
	const customDateFormat = "2006-01-02 15:04"
	var parsedDate time.Time

	// try parsing with the custom format
	parsedDate, err := time.Parse(customDateFormat, date)
	if err != nil {
		// try RFC3339
		parsedDate, err = time.Parse(time.RFC3339, date)
		if err != nil {
			return "", fmt.Errorf("Failed to parse date; ensure the format is correct")
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
func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Message: message,
	})
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
	return ok && (role == "REGISTERED" || role == "ADMIN")
}

// Get user UUID from the request context.
func getUserIDFromContext(ctx context.Context) (string, error) {
	claims, err := middlewares.GetClaimsFromContext(ctx)
	if err != nil {
		return "", err
	}
	userId, ok := claims["userID"].(string)
	if !ok {
		return "", fmt.Errorf("Unable to extract userId from request context")
	}
	return userId, nil
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
