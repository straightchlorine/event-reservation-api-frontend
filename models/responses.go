package models

import (
	"time"

	"github.com/google/uuid"
)

// Standardized response for errors.
type ErrorResponse struct {
	Message string `json:"message" example:"An error occurred"`
}

// Standardized response for successful operations.
type SuccessResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// Standardized response for successful operations involving creating objects.
type SuccessResponseCreate struct {
	Message string `json:"message" example:"Object created successfully"`
	ID      int    `json:"id"      example:"1"`
}

// Standardized response for successful operations involving creating objects with UUID.
type SuccessResponseCreateUUID struct {
	Message string `json:"message" example:"Object created successfully"`
	UUID    string `json:"id"      example:"123e4567-e89b-12d3-a456-426614174000"`
}

// User ID and username.
type UserUsernameID struct {
	ID       string `json:"id"       example:"123e4567-e89b-12d3-a456-426614174000"`
	Username string `json:"username" example:"root"`
}

// Response after a successful login.
type LoginResponse struct {
	Token   string         `json:"token" example:"jwt-token-string"`
	Expires int64          `json:"exp"   example:"1683649261"`
	User    UserUsernameID `json:"user"`
}

// Event, as it's returned to the user.
type EventResponse struct {
	ID               int              `json:"id"                example:"1"`
	Name             string           `json:"name"              example:"Champions League Final"`
	Price            float64          `json:"price"             example:"99.99"`
	AvailableTickets int              `json:"available_tickets" example:"15000"`
	Date             time.Time        `json:"date"              example:"2024-12-31T20:00:00Z"`
	Location         LocationResponse `json:"location"`
}

// Collection of events.
type EventsResponse struct {
	Events []EventResponse `json:"events"`
}

// Location, as it's returned to the user.
type LocationResponse struct {
	ID       int    `json:"id"       example:"101"`
	Country  string `json:"country"  example:"England"`
	Address  string `json:"address"  example:"Wembley Park, London"`
	Stadium  string `json:"stadium"  example:"Wembley Stadium"`
	Capacity int    `json:"capacity" example:"90000"`
}

// Collection of locations.
type LocationsResponse struct {
	Locations []LocationResponse `json:"locations"`
}

// User response, as it's returned to the user.
type UserResponse struct {
	ID        uuid.UUID `json:"id"                   example:"123e4567-e89b-12d3-a456-426614174000"`
	Name      string    `json:"name"                 example:"John"`
	Surname   string    `json:"surname"              example:"Doe"`
	Username  string    `json:"username"             example:"johndoe"`
	Email     string    `json:"email"                example:"johndoe@example.com"`
	LastLogin time.Time `json:"last_login,omitempty" example:"2024-12-01T15:30:00Z"`
	CreatedAt time.Time `json:"created_at"           example:"2024-01-01T10:00:00Z"`
	RoleName  string    `json:"role_id"              example:"admin"`
	IsActive  bool      `json:"is_active"            example:"true"`
}

// Collection of users.
type UsersResponse struct {
	Users []UserResponse `json:"users"`
}

// Ticket, as it's returned to the user.
type TicketResponse struct {
	ID     string  `json:"id"     example:"abc123"`
	Type   string  `json:"type"   example:"STANDARD"`
	Price  float64 `json:"price"  example:"150.00"`
	Status string  `json:"status" example:"available"`
}

// User's ticket response.
type UserTicketResponse struct {
	ID            string        `json:"id"             example:"ticket123"`
	Type          string        `json:"type"           example:"STANDARD"`
	Price         float64       `json:"price"          example:"50.00"`
	Status        string        `json:"status"         example:"SOLD"`
	ReservationID string        `json:"reservation_id" example:"res123"`
	Event         EventResponse `json:"event"`
}

// User's collection of tickets.
type UserTicketsResponse struct {
	UserID  string               `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Tickets []UserTicketResponse `json:"tickets"`
}

// Reservation, as it's returned to the user.
type ReservationResponse struct {
	ID           string           `json:"id"            example:"res123"`
	Username     string           `json:"user"          example:"johndoe"`
	CreatedAt    time.Time        `json:"created_at"    example:"2024-12-01T15:30:00Z"`
	TotalTickets int              `json:"total_tickets" example:"5"`
	Status       string           `json:"status"        example:"CONFIRMED"`
	Event        EventResponse    `json:"event"`
	Tickets      []TicketResponse `json:"tickets"`
}

// Collection of reservations.
type ReservationsResponse struct {
	Reservations []ReservationResponse `json:"reservations"`
}

// Response for tickets under a reservation.
type ReservationTicketsResponse struct {
	ReservationID string           `json:"reservation_id" example:"res123"`
	UserID        string           `json:"user"           example:"johndoe"`
	Tickets       []TicketResponse `json:"tickets"`
}
