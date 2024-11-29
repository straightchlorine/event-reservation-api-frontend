package models

// Standardized response for errors.
type ErrorResponse struct {
	Message string `json:"message" example:"Error message"`
	Status  int    `json:"status"  example:"400"`
}

// Standardized response for successful operations.
type SuccessResponse struct {
	Message string `json:"message" example:"Error message"`
	Status  int    `json:"status"  example:"400"`
}

// Standardized response for successful involving creating objects.
type SuccessResponseCreate struct {
	Message string `json:"message" example:"Error message"`
	Status  int    `json:"status"  example:"400"`
	ID      string `json:"id"      example:"123e4567-e89b-12d3-a456-426614174000"`
}

// User ID and username.
type UserUsernameID struct {
	ID       string `json:"id"       example:"123e4567-e89b-12d3-a456-426614174000"`
	Username string `json:"username" example:"root"`
}

// Response after a successful login.
type LoginResponse struct {
	Token   string         `json:"token" example:"jwt-token-string"`
	Expires int64          `json:"exp"   example:"12313123"`
	User    UserUsernameID `json:"user"`
}
