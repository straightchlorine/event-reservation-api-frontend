package middlewares

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Retrieve an environment variable as an integer or returns a default value.
func getEnvAsInt(key string, defaultValue int) (int, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue, fmt.Errorf("invalid value for %s: %v", key, err)
	}
	return value, nil
}

// Create a random 32-byte secret and encode it in Base64.
func generateRandomSecret() (string, error) {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate random secret: %v", err)
	}
	return base64.StdEncoding.EncodeToString(secret), nil
}

// Creates a JWT token with user claims.
func GenerateJWT(userID string, role string, secret string) (string, int64, error) {
	// get the token validity duration from env variable, otherwise 24 hours
	validHours, err := getEnvAsInt("TOKEN_VALID_HOURS", 24)
	if err != nil {
		log.Printf("Invalid TOKEN_VALID_HOURS, defaulting to 24 hours: %v", err)
	}

	expirationTime := time.Now().Add(time.Hour * time.Duration(validHours)).Unix()

	// generate the claims
	claims := jwt.MapClaims{
		"userID": userID,
		"role":   role,
		"exp":    expirationTime,
	}

	// create the JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expirationTime, nil
}

// Validate the JWT token from the Authorization header
func ValidateJWT(tokenString string, secret string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(secret), nil
	})
}

// Initialize JWT secret from the environment or generate one as fallback
func InitJWTSecret() string {
	// check if the secret is set in the environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret != "" {
		return jwtSecret
	}

	// generate a random secret if none is set
	randomSecret, err := generateRandomSecret()
	if err != nil {
		log.Fatalf("Failed to generate random JWT secret: %v", err)
	}

	log.Println(
		"WARNING: JWT_SECRET not set. Generating a random secret. Tokens will not be consistent across restarts.",
	)
	return randomSecret
}
