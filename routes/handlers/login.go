package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest represents the expected login payload.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the structure of the response for a successful login.
type LoginResponse struct {
	Token string `json:"token"`
}

// loginHandler authenticates the user and returns a JWT.
func LoginHandler(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var loginReq LoginRequest

		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// Query the database for user details
		var userID int
		var hashedPassword, role string
		query := `SELECT u.id, u.password_hash, r.name AS role_name FROM users u JOIN roles r ON u.role_id = r.id WHERE u.username = $1`
		err := pool.QueryRow(context.Background(), query, loginReq.Username).Scan(&userID, &hashedPassword, &role)
		if err != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Validate the provided password against the hashed password
		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(loginReq.Password)); err != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Generate a JWT for the authenticated user
		claims := jwt.MapClaims{
			"userID": userID,
			"role":   role,
			"exp":    time.Now().Add(24 * time.Hour).Unix(), // Token expires in 24 hours
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtSecret))

		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Send the token back to the client
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Authorization", "Bearer "+tokenString)
		json.NewEncoder(w).Encode(LoginResponse{Token: tokenString})
	}
}
