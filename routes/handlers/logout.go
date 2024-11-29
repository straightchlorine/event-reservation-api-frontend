package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/middlewares"
	"event-reservation-api/models"
)

// Logout handler facilitates the logout process.
//
//	@Summary		Logout from the API (admin or registered user)
//	@Description	Invalidate currently used JWT token.
//	@ID				api.logout
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	models.LoginResponse	"Successfully logged out"
//	@Failure		401	{object}	models.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	models.ErrorResponse	"Internal Server Error"
//	@Router			/logout [post]
func LogoutHandler(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// extract the claims from the token
		tokenString, err := middlewares.ExtractToken(r)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "You are not logged in.")
			return
		}

		// extract the claims
		claims, err := middlewares.GetValidatedClaims(tokenString, jwtSecret)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Failed to validate the token.")
			return
		}

		// get the expiration time from the claims
		expirationTime, ok := claims["exp"].(float64)
		if !ok {
			writeErrorResponse(
				w,
				http.StatusUnauthorized,
				"Unable to extract token expiration time.",
			)
			return
		}

		// invalidate current token
		if err := invalidateToken(pool, tokenString, expirationTime); err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		// respond with a success message
		writeJSONResponse(
			w,
			http.StatusOK,
			models.SuccessResponse{Message: "Logged out successfully."},
		)
	}
}

// Invalidate the token by adding it to the blacklist.
func invalidateToken(pool *pgxpool.Pool, tokenString string, expirationTime float64) error {
	query := `INSERT INTO token_blacklist (token, expires_at) VALUES ($1, $2)`
	if _, err := pool.Exec(
		context.Background(),
		query,
		tokenString,
		time.Unix(int64(expirationTime), 0),
	); err != nil {
		return fmt.Errorf("Failed to invalidate the token.")
	}
	return nil
}
