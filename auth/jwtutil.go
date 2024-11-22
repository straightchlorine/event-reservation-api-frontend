package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateSecret creates a random secret key for JWT signing
func GenerateSecret() string {
	// Generate a random byte slice
	secret := make([]byte, 32) // 32 bytes = 256 bits

	// Fill the byte slice with random data
	_, err := rand.Read(secret)
	if err != nil {
		// If there's an error generating the random bytes, return a default string
		// It's important to handle this error in a production environment.
		return "defaultsecretkey" // Ideally, this would not be used in production
	}

	// Encode the random bytes as a base64 string (can be used as a secret)
	encodedSecret := base64.StdEncoding.EncodeToString(secret)

	// Return the base64-encoded secret
	return encodedSecret
}

// GenerateJWT creates a JWT token with user claims
func GenerateJWT(userID int, role string, secret string) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"role":   role,
		"exp":    time.Now().Add(time.Hour * 1).Unix(), // Expire after 1 hour
	}

	// Create the JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and return the token
	return token.SignedString([]byte(secret))
}

// ValidateJWT validates the JWT token from the Authorization header
func ValidateJWT(tokenString string, secret string) (*jwt.Token, error) {
	// Parse the JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

// Initialize JWT Secret (from environment or generate if not present)
func InitJWTSecret() string {
	// Check if the JWT_SECRET environment variable exists
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Generate a random JWT secret if not provided in the environment
		jwtSecret = GenerateSecret()
		log.Printf("JWT_SECRET was not set, using generated secret: %s", jwtSecret)
	}
	return jwtSecret
}
