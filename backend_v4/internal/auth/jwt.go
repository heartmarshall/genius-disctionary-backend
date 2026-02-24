package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTManager handles JWT access token generation and validation,
// plus refresh token generation and hashing.
type JWTManager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
}

// NewJWTManager creates a new JWT manager.
// secret must be at least 32 characters for HS256 security.
func NewJWTManager(secret string, issuer string, accessTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:    []byte(secret),
		issuer:    issuer,
		accessTTL: accessTTL,
	}
}

// accessClaims extends standard JWT claims with the user's role.
type accessClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role,omitempty"`
}

// GenerateAccessToken creates a signed HS256 JWT with user ID as subject and role as a custom claim.
func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, role string) (string, error) {
	now := time.Now()
	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Role: role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}

// ValidateAccessToken parses and validates a JWT access token.
// Returns the user ID and role if valid.
func (m *JWTManager) ValidateAccessToken(tokenString string) (uuid.UUID, string, error) {
	if tokenString == "" {
		return uuid.Nil, "", fmt.Errorf("token is empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &accessClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return uuid.Nil, "", fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*accessClaims)
	if !ok || !token.Valid {
		return uuid.Nil, "", fmt.Errorf("invalid token claims")
	}

	if claims.Issuer != m.issuer {
		return uuid.Nil, "", fmt.Errorf("invalid issuer: expected %s, got %s", m.issuer, claims.Issuer)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid subject UUID: %w", err)
	}

	return userID, claims.Role, nil
}

// GenerateRefreshToken creates a cryptographically random refresh token.
// Returns both the raw token (to send to client) and its SHA-256 hash (to store in DB).
func (m *JWTManager) GenerateRefreshToken() (raw string, hash string, err error) {
	// Generate 32 random bytes
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate random bytes: %w", err)
	}

	// Encode to base64 for the raw token
	raw = base64.RawURLEncoding.EncodeToString(b)

	// Hash for storage
	hash = HashToken(raw)

	return raw, hash, nil
}

// HashToken computes the SHA-256 hash of a token and returns it as a hex string.
// This is used to hash refresh tokens before storing them in the database.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
