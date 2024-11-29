package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"event-reservation-api/middlewares"
	"event-reservation-api/models"
)

// Login handler godoc
//
//	@Summary			Login to the API.
//	@Description	Pass username and password to authenticate and get a JWT token.
//	@ID						api.login
//	@Tags					auth
//	@Accept				json
//	@Produce			json
//	@Param				body		body		models.LoginRequest		true	"Login credentials"
//	@Success			200		{object}	models.LoginResponse	"Successfully logged in"
//	@Failure			400		{object}	models.ErrorResponse	"Bad Request"
//	@Failure			401		{object}	models.ErrorResponse	"Unauthorized"
//	@Failure			500		{object}	models.ErrorResponse	"Internal Server Error"
//	@Router				/login [post]
func LoginHandler(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// parse login request
		var loginReq models.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload.")
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
			writeErrorResponse(w, http.StatusNotFound, "User not found.")
			return
		}

		// check if the password matches the hash
		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(loginReq.Password)); err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Invalid username or password.")
			return
		}

		// get the token validity duration from env variable, otherwise 24 hours
		tokenString, exp, err := middlewares.GenerateJWT(userID, role, jwtSecret)
		if err != nil {
			writeErrorResponse(
				w,
				http.StatusInternalServerError,
				"Failed to generate access token.",
			)
			return
		}

		// send the token back to client
		writeTokenResponse(w, tokenString, exp, userID, loginReq.Username)
	}
}

func writeTokenResponse(
	w http.ResponseWriter,
	token string,
	exp int64,
	userID string,
	username string,
) {
	w.Header().Set("Content-Type", "application/json")

	// append user details to the reponse
	user := models.UserUsernameID{ID: userID, Username: username}
	json.NewEncoder(w).Encode(models.LoginResponse{Token: token, Expires: exp, User: user})
}
