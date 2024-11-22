package models

import (
	"time"

	"github.com/google/uuid"
)

// Role represents user roles in the system
type Role struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// User represents the user account information
type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Surname      string    `json:"surname"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	LastLogin    time.Time `json:"last_login,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
	RoleID       int       `json:"role_id"`
	IsActive     bool      `json:"is_active"`
}

// UserAuthLog represents the user authentication log
type UserAuthLog struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	LoginTime   time.Time `json:"login_time"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	LoginStatus bool      `json:"login_status"`
}

// Location represents event venues
type Location struct {
	ID       int    `json:"id"`
	Stadium  string `json:"stadium"`
	Address  string `json:"address"`
	Country  string `json:"country"`
	Capacity int    `json:"capacity"`
}

// Event represents a specific event
type Event struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Date             time.Time `json:"date"`
	LocationID       int       `json:"location_id"`
	AvailableTickets int       `json:"available_tickets"`
}

// ReservationStatus represents different states of a reservation
type ReservationStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Reservation represents an order of at least one ticket
type Reservation struct {
	ID            uuid.UUID `json:"id"`
	PrimaryUserID int       `json:"primary_user_id"`
	EventID       int       `json:"event_id"`
	CreatedAt     time.Time `json:"created_at"`
	TotalTickets  int       `json:"total_tickets"`
	StatusID      int       `json:"status_id"`
}

// TicketType represents different types of tickets
type TicketType struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Discount    float64 `json:"discount"`
	Description string  `json:"description"`
}

// TicketStatus represents different states of a ticket
type TicketStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Ticket represents an individual ticket
type Ticket struct {
	ID            int       `json:"id"`
	EventID       int       `json:"event_id"`
	ReservationID uuid.UUID `json:"reservation_id"`
	Price         float64   `json:"price"`
	TypeID        int       `json:"type_id"`
	StatusID      int       `json:"status_id"`
	SeatNumber    string    `json:"seat_number,omitempty"`
}

// PaymentStatus represents different states of a payment
type PaymentStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Payment represents a payment for a reservation
type Payment struct {
	ID           int       `json:"id"`
	GroupOrderID uuid.UUID `json:"reservation_id"`
	StatusID     int       `json:"status_id"`
	TotalAmount  float64   `json:"total_amount"`
	PaymentDate  time.Time `json:"payment_date"`
}

// Permission represents system-wide permissions
type Permission struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
