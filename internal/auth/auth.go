package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// DefaultTokenExpiry is the default token expiration time in seconds
	DefaultTokenExpiry = 3600 // 1 hour
	// MinTokenExpiry is the minimum allowed token expiration time
	MinTokenExpiry = 60 // 1 minute
	// MaxTokenExpiry is the maximum allowed token expiration time
	MaxTokenExpiry = 86400 // 24 hours
)

var (
	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when the token has expired
	ErrExpiredToken = errors.New("token expired")
	// ErrEmptyToken is returned when the token is empty
	ErrEmptyToken = errors.New("empty token")
	// ErrMissingSecret is returned when the JWT secret is not configured
	ErrMissingSecret = errors.New("JWT secret not configured")
)

// Claims represents JWT authentication claims
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Manager handles authentication
type Manager struct {
	secret string
}

// NewManager creates a new authentication manager
// It reads the JWT secret from the JWT_SECRET environment variable
// If not set, it returns an error
func NewManager() (*Manager, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, ErrMissingSecret
	}

	// Validate secret length (minimum 32 characters for HS256)
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT secret must be at least 32 characters long, got %d", len(secret))
	}

	return &Manager{
		secret: secret,
	}, nil
}

// NewManagerWithSecret creates a new authentication manager with a provided secret
// This is useful for testing or when you want to inject the secret directly
func NewManagerWithSecret(secret string) (*Manager, error) {
	if secret == "" {
		return nil, ErrMissingSecret
	}

	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT secret must be at least 32 characters long, got %d", len(secret))
	}

	return &Manager{
		secret: secret,
	}, nil
}

// GenerateToken generates a JWT authentication token
// expiresIn is the token expiration time in seconds
func (m *Manager) GenerateToken(userID int, username string, expiresIn int) (string, error) {
	// Validate expiration time
	if expiresIn < MinTokenExpiry {
		expiresIn = MinTokenExpiry
	}
	if expiresIn > MaxTokenExpiry {
		expiresIn = MaxTokenExpiry
	}

	// Create claims
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expiresIn) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "open-db9",
			Subject:   fmt.Sprintf("user:%d", userID),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(m.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT authentication token and returns the claims
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrEmptyToken
	}

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshToken refreshes an authentication token
// It validates the existing token and generates a new one with extended expiration
func (m *Manager) RefreshToken(tokenString string, expiresIn int) (string, error) {
	// Validate existing token
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("failed to validate token for refresh: %w", err)
	}

	// Generate new token with same user info
	newToken, err := m.GenerateToken(claims.UserID, claims.Username, expiresIn)
	if err != nil {
		return "", fmt.Errorf("failed to generate new token: %w", err)
	}

	return newToken, nil
}

// GenerateAnonymousToken generates a JWT token for anonymous users
// The tenantID is used as the subject and the token includes an anonymous claim
func (m *Manager) GenerateAnonymousToken(tenantID string) (string, error) {
	if tenantID == "" {
		return "", fmt.Errorf("tenant ID is required")
	}

	// Create claims for anonymous user
	now := time.Now()
	claims := &Claims{
		UserID:   0, // Anonymous users have no user ID
		Username: "anonymous",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)), // 24 hours for anonymous tokens
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "open-db9",
			Subject:   fmt.Sprintf("anonymous:%s", tenantID),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(m.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign anonymous token: %w", err)
	}

	return tokenString, nil
}
