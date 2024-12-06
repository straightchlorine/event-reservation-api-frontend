package middlewares

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// context key type
type ContextKey string

// JWT claims stored in the context
const UserClaimsKey ContextKey = "userClaims"

// Routine for token cleanup.
func StartTokenCleanupTask(pool *pgxpool.Pool, interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			if err := DeleteExpiredTokens(pool); err != nil {
				fmt.Printf("Error deleting expired tokens: %v\n", err)
			}
		}
	}()
}

// Delete expire tokens from the blacklist.
func DeleteExpiredTokens(pool *pgxpool.Pool) error {
	log.Println("Deleting expired tokens...")
	query := `DELETE FROM token_blacklist WHERE expires_at < $1`
	_, err := pool.Exec(context.Background(), query, time.Now())
	if err != nil {
		return err
	}
	return nil
}

func TokenValidation(
	pool *pgxpool.Pool,
	jwtSecret string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// token extraction
			tokenString, err := ExtractToken(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// check if the token is in the blacklist.
			var exists bool
			query := `SELECT EXISTS (SELECT 1 FROM token_blacklist WHERE token = $1)`
			err = pool.QueryRow(context.Background(), query, tokenString).Scan(&exists)
			if err != nil || exists {
				http.Error(w, "Token is invalid", http.StatusUnauthorized)
				return
			}

			// validate the token.
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, http.ErrAbortHandler
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// pass the request to the next handler.
			next.ServeHTTP(w, r)
		})
	}
}

// Validate JWT tokens using HMAC and add claims to the request context.
func RequireAuth(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// token extraction
			tokenString, err := ExtractToken(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// extract and validate the claims
			claims, err := GetValidatedClaims(tokenString, jwtSecret)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// add claims to the request context
			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Extract the JWT token from the Authorization header.
func ExtractToken(r *http.Request) (string, error) {
	// extract the authentication header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Missing Authorization header")
	}

	// remove the prefix
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return "", fmt.Errorf("Malformed Authorization header")
	}

	return tokenString, nil
}

// Parse and validate the JWT token using the provided secret.
func GetValidatedClaims(tokenString, jwtSecret string) (jwt.MapClaims, error) {
	token, err := ValidateJWT(tokenString, jwtSecret)

	if err != nil {
		return nil, fmt.Errorf("invalid or expired token: %v", err)
	}

	// extract the claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// Retrieve JWT claims from the request context.
func GetClaimsFromContext(ctx context.Context) (jwt.MapClaims, error) {
	claims, ok := ctx.Value(UserClaimsKey).(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("no valid claims in context")
	}
	return claims, nil
}
