package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ContextKey string

// JWT claims stored in the context
const UserClaimsKey ContextKey = "userClaims"

/*
Validate JWT tokens using HMAC and add claims to the request context.

Arguments:

	jwtSecret: Secret key used to sign the JWT token.

Returns:

	http.Handler: Middleware function, ensuring authorization during routing.
*/
func RequireAuth(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := extractToken(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims, err := validateJWT(tokenString, jwtSecret)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// Add claims to the request context
			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

/*
Extract the JWT token from the Authorization header.

Arguments:

	r: HTTP request object.

Returns:

	token and error (nil if successful).
*/
func extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Missing Authorization header")
	}

	// Remove the "Bearer " prefix
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return "", fmt.Errorf("Malformed Authorization header")
	}

	return tokenString, nil
}

/*
Parse and validate the JWT token using the provided secret.

Arguments:

	tokenString: JWT token string.
	jwtSecret: Secret key used to sign the JWT token.

Returns:

	jwt.MapClaims and error (nil if successful).
*/
func validateJWT(tokenString, jwtSecret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid or expired token: %v", err)
	}

	// Extract and verify claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

/*
Retrieve JWT claims from the request context.

Arguments:

	ctx: Request context.

Returns:

	jwt.MapClaims and error (nil if successful).
*/
func GetClaimsFromContext(ctx context.Context) (jwt.MapClaims, error) {
	claims, ok := ctx.Value(UserClaimsKey).(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("no valid claims in context")
	}
	return claims, nil
}
