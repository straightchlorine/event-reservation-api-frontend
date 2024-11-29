package models

// Expected login payload.
type LoginRequest struct {
	Username string `json:"username" example:"root"`
	Password string `json:"password" example:"root"`
}

// Expected create location payload.
type CreateLocationRequest struct {
	Address  string `json:"address"`
	Capacity int    `json:"capacity,omitempty"`
	Country  string `json:"country,omitempty"`
	Stadium  string `json:"stadium"`
}

// Expected create event payload.
type CreateEventRequest struct {
	Name             string                `json:"name"`
	Date             string                `json:"date"`
	AvailableTickets int                   `json:"available_tickets"`
	Price            float64               `json:"price"`
	Location         CreateLocationRequest `json:"location"`
}

type CreateUserRequest struct {
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	RoleName string `json:"role_name"`
	IsActive bool   `json:"is_active"`
}

// Expected update location payload.
type UpdateLocationRequest struct {
	Address  *string `json:"address,omitempty"`
	Capacity *int    `json:"capacity,omitempty"`
	Country  *string `json:"country,omitempty"`
	Stadium  *string `json:"stadium,omitempty"`
}

// Expected update event payload.
type UpdateEventRequest struct {
	AvailableTickets *int                   `json:"available_tickets,omitempty"`
	Date             *string                `json:"date,omitempty"`
	Name             *string                `json:"name,omitempty"`
	Price            *float64               `json:"price, omitempty"`
	Location         *UpdateLocationRequest `json:"location,omitempty"`
}

type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty"`
	Surname  *string `json:"surname,omitempty"`
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
	Email    *string `json:"email,omitempty"`
	RoleName *string `json:"role_name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}
