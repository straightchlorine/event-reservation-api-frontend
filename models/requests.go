package models

// Expected login payload.
type LoginRequest struct {
	Username string `json:"username" example:"user123"`
	Password string `json:"password" example:"securepassword"`
}

// Expected create location payload.
type CreateLocationRequest struct {
	Address  string `json:"address"            example:"123 Main St"`
	Capacity int    `json:"capacity,omitempty" example:"50000"`
	Country  string `json:"country,omitempty"  example:"USA"`
	Stadium  string `json:"stadium"            example:"National Stadium"`
}

// Expected create event payload.
type CreateEventRequest struct {
	Name             string                `json:"name"              example:"Champions League Final"`
	Date             string                `json:"date"              example:"2024-12-31T20:00:00Z"`
	AvailableTickets int                   `json:"available_tickets" example:"20000"`
	Price            float64               `json:"price"             example:"99.99"`
	Location         CreateLocationRequest `json:"location"`
}

// Expected create user payload.
type CreateUserRequest struct {
	Name     string `json:"name"      example:"John"`
	Surname  string `json:"surname"   example:"Doe"`
	Username string `json:"username"  example:"johndoe"`
	Email    string `json:"email"     example:"johndoe@example.com"`
	Password string `json:"password"  example:"strongpassword"`
	RoleName string `json:"role_name" example:"user"`
	IsActive bool   `json:"is_active" example:"true"`
}

// Structure of a valid payload to create a reservation.
type CreateReservationPayload struct {
	EventID int `json:"event_id" example:"101"`
	Tickets []struct {
		Type string `json:"type" example:"STANDARD"`
	} `json:"tickets"`
}

// Structure of a valid request to the database.
type ReservationRequest struct {
	UserID       string `json:"user_id"       example:"123e4567-e89b-12d3-a456-426614174000"`
	EventID      int    `json:"event_id"      example:"101"`
	TotalTickets int    `json:"total_tickets" example:"3"`
	StatusID     int    `json:"status_id"     example:"1"`
	Tickets      []struct {
		Type string `json:"type" example:"Regular"`
	} `json:"tickets"`
}

// Expected update location payload.
type UpdateLocationRequest struct {
	Address  *string `json:"address,omitempty"  example:"456 Elm St"`
	Capacity *int    `json:"capacity,omitempty" example:"75000"`
	Country  *string `json:"country,omitempty"  example:"Canada"`
	Stadium  *string `json:"stadium,omitempty"  example:"Maple Leaf Stadium"`
}

// Expected update event payload.
type UpdateEventRequest struct {
	AvailableTickets *int                   `json:"available_tickets,omitempty" example:"15000"`
	Date             *string                `json:"date,omitempty"              example:"2024-12-25T18:00:00Z"`
	Name             *string                `json:"name,omitempty"              example:"Christmas Special"`
	Price            *float64               `json:"price,omitempty"             example:"49.99"`
	Location         *UpdateLocationRequest `json:"location,omitempty"`
}

// Expected update user payload.
type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty"      example:"Jane"`
	Surname  *string `json:"surname,omitempty"   example:"Smith"`
	Username *string `json:"username,omitempty"  example:"janesmith"`
	Password *string `json:"password,omitempty"  example:"newsecurepassword"`
	Email    *string `json:"email,omitempty"     example:"janesmith@example.com"`
	RoleName *string `json:"role_name,omitempty" example:"admin"`
	IsActive *bool   `json:"is_active,omitempty" example:"false"`
}
