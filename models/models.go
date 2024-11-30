package models

import (
	"time"

	"github.com/google/uuid"
)

// Token blacklist entry
type BlacklistEntry struct {
	Token     string
	ExpiresAt time.Time
}

type LocationUpdatePayload struct {
	Stadium  *string `json:"stadium,omitempty"`
	Address  *string `json:"address,omitempty"`
	Country  *string `json:"country,omitempty"`
	Capacity *int    `json:"capacity,omitempty"`
}

// Role table record
type Role struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// User table record
type User struct {
	ID           uuid.UUID `json:"id"`
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

// UserAuthLog table record
type UserAuthLog struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	LoginTime   time.Time `json:"login_time"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	LoginStatus bool      `json:"login_status"`
}

// Location table record
type Location struct {
	ID       int    `json:"id"`
	Stadium  string `json:"stadium"`
	Address  string `json:"address"`
	Country  string `json:"country"`
	Capacity int    `json:"capacity"`
}

// Event table record
type Event struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Date             time.Time `json:"date"`
	Price            float64   `json:"price"`
	LocationID       int       `json:"location_id"`
	AvailableTickets int       `json:"available_tickets"`
}

// ReservationStatus table record
type ReservationStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Reservation table record
type Reservation struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	EventID      int       `json:"event_id"`
	CreatedAt    time.Time `json:"created_at"`
	TotalTickets int       `json:"total_tickets"`
	StatusID     int       `json:"status_id"`
}

// TicketType table record
type TicketType struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Discount    float64 `json:"discount"`
	Description string  `json:"description"`
}

// TicketStatus table record
type TicketStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Ticket table record
type Ticket struct {
	ID            int       `json:"id"`
	ReservationID uuid.UUID `json:"reservation_id"`
	Price         float64   `json:"price"`
	TypeID        int       `json:"type_id"`
	StatusID      int       `json:"status_id"`
}

// PaymentStatus table record
type PaymentStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Payment table record
type Payment struct {
	ID           int       `json:"id"`
	GroupOrderID uuid.UUID `json:"reservation_id"`
	StatusID     int       `json:"status_id"`
	TotalAmount  float64   `json:"total_amount"`
	PaymentDate  time.Time `json:"payment_date"`
}

// Permission table record
type Permission struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
