package models

// Expected login payload.
type LoginRequest struct {
	Username string `json:"username" example:"root"`
	Password string `json:"password" example:"root"`
}
