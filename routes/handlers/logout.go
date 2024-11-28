package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/middlewares"
)

// Invalidate current token by on logout.
func LogoutHandler(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// extract the claims from the token
		tokenString, err := middlewares.ExtractToken(r)
		if err != nil {
			handleError(w, http.StatusUnauthorized, "You are not logged in.", nil)
			return
		}

		// extract the claims
		claims, err := middlewares.GetValidatedClaims(tokenString, jwtSecret)
		if err != nil {
			handleError(w, http.StatusUnauthorized, "Failed to validate token", err)
			return
		}

		// get the expiration time from the claims
		expirationTime, ok := claims["exp"].(float64)
		if !ok {
			handleError(w, http.StatusUnauthorized, "Invalid token expiration", nil)
			return
		}

		// insert the token into the blacklist
		query := `INSERT INTO token_blacklist (token, expires_at) VALUES ($1, $2)`
		if _, err := pool.Exec(
			context.Background(),
			query,
			tokenString,
			time.Unix(int64(expirationTime), 0),
		); err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to invalidate token", err)
			return
		}

		// respond with a success message
		writeJSONResponse(w, http.StatusOK, map[string]string{
			"message": "Logged out successfully",
		})
	}
}
