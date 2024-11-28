package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"event-reservation-api/middlewares"
)

// Expected login payload.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Response after a successful login.
type LoginResponse struct {
	Token string `json:"token"`
}

// Authenticate the user and return a JWT.
func LoginHandler(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var loginReq LoginRequest

		// parse login request
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// query the database for user details
		var userID string
		var hashedPassword, role string
		query := `
			SELECT u.id, u.password_hash, r.name
			FROM users u
			JOIN roles r ON u.role_id = r.id
			WHERE u.username = $1`
		if err := pool.QueryRow(
			context.Background(), query, loginReq.Username,
		).Scan(&userID, &hashedPassword, &role); err != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// check if the password matches the hash
		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(loginReq.Password)); err != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// get the token validity duration from env variable, otherwise 24 hours
		tokenString, err := middlewares.GenerateJWT(userID, role, jwtSecret)

		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// send the token back to client
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Authorization", "Bearer "+tokenString)
		json.NewEncoder(w).Encode(LoginResponse{Token: tokenString})
	}
}
