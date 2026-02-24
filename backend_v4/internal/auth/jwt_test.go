package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTManager_GenerateAndValidate_Success(t *testing.T) {
	secret := "test-secret-at-least-32-chars-long-for-security"
	issuer := "myenglish-test"
	ttl := 15 * time.Minute

	manager := NewJWTManager(secret, issuer, ttl)
	userID := uuid.New()

	// Generate token
	token, err := manager.GenerateAccessToken(userID, "user")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Validate token
	validatedID, role, err := manager.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if validatedID != userID {
		t.Errorf("expected userID %s, got %s", userID, validatedID)
	}
	if role != "user" {
		t.Errorf("expected role 'user', got %q", role)
	}
}

func TestJWTManager_GenerateAndValidate_AdminRole(t *testing.T) {
	secret := "test-secret-at-least-32-chars-long-for-security"
	issuer := "myenglish-test"
	ttl := 15 * time.Minute

	manager := NewJWTManager(secret, issuer, ttl)
	userID := uuid.New()

	token, err := manager.GenerateAccessToken(userID, "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, role, err := manager.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if role != "admin" {
		t.Errorf("expected role 'admin', got %q", role)
	}
}

func TestJWTManager_ValidateAccessToken_Expired(t *testing.T) {
	secret := "test-secret-at-least-32-chars-long-for-security"
	issuer := "myenglish-test"
	ttl := -1 * time.Hour // Already expired

	manager := NewJWTManager(secret, issuer, ttl)
	userID := uuid.New()

	token, err := manager.GenerateAccessToken(userID, "user")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Should fail validation due to expiry
	_, _, err = manager.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if !strings.Contains(err.Error(), "expired") && !strings.Contains(err.Error(), "parse token") {
		t.Errorf("expected expiry-related error, got: %v", err)
	}
}

func TestJWTManager_ValidateAccessToken_InvalidSignature(t *testing.T) {
	secret1 := "test-secret-at-least-32-chars-long-for-security"
	secret2 := "different-secret-32-chars-long-for-security!!"
	issuer := "myenglish-test"
	ttl := 15 * time.Minute

	manager1 := NewJWTManager(secret1, issuer, ttl)
	manager2 := NewJWTManager(secret2, issuer, ttl)
	userID := uuid.New()

	// Generate with manager1
	token, err := manager1.GenerateAccessToken(userID, "user")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Validate with manager2 (different secret)
	_, _, err = manager2.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
}

func TestJWTManager_ValidateAccessToken_Malformed(t *testing.T) {
	secret := "test-secret-at-least-32-chars-long-for-security"
	issuer := "myenglish-test"
	ttl := 15 * time.Minute

	manager := NewJWTManager(secret, issuer, ttl)

	malformedTokens := []string{
		"not.a.jwt",
		"invalid-token",
		"header.payload", // Missing signature
	}

	for _, token := range malformedTokens {
		_, _, err := manager.ValidateAccessToken(token)
		if err == nil {
			t.Errorf("expected error for malformed token %q, got nil", token)
		}
	}
}

func TestJWTManager_ValidateAccessToken_WrongIssuer(t *testing.T) {
	secret := "test-secret-at-least-32-chars-long-for-security"
	issuer1 := "myenglish-test"
	issuer2 := "wrong-issuer"
	ttl := 15 * time.Minute

	manager1 := NewJWTManager(secret, issuer1, ttl)
	manager2 := NewJWTManager(secret, issuer2, ttl)
	userID := uuid.New()

	// Generate with manager1 (issuer1)
	token, err := manager1.GenerateAccessToken(userID, "user")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Validate with manager2 (issuer2)
	_, _, err = manager2.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("expected error for wrong issuer, got nil")
	}
	if !strings.Contains(err.Error(), "invalid issuer") {
		t.Errorf("expected 'invalid issuer' error, got: %v", err)
	}
}

func TestJWTManager_ValidateAccessToken_EmptyString(t *testing.T) {
	secret := "test-secret-at-least-32-chars-long-for-security"
	issuer := "myenglish-test"
	ttl := 15 * time.Minute

	manager := NewJWTManager(secret, issuer, ttl)

	_, _, err := manager.ValidateAccessToken("")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected 'empty' error, got: %v", err)
	}
}

func TestJWTManager_GenerateRefreshToken_Uniqueness(t *testing.T) {
	secret := "test-secret-at-least-32-chars-long-for-security"
	issuer := "myenglish-test"
	ttl := 15 * time.Minute

	manager := NewJWTManager(secret, issuer, ttl)

	// Generate multiple tokens
	tokens := make(map[string]bool)
	hashes := make(map[string]bool)

	for range 100 {
		raw, hash, err := manager.GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken failed: %v", err)
		}
		if raw == "" || hash == "" {
			t.Fatal("expected non-empty raw and hash")
		}

		// Check uniqueness
		if tokens[raw] {
			t.Errorf("duplicate raw token: %s", raw)
		}
		if hashes[hash] {
			t.Errorf("duplicate hash: %s", hash)
		}

		tokens[raw] = true
		hashes[hash] = true
	}
}

func TestJWTManager_GenerateRefreshToken_HashMatches(t *testing.T) {
	secret := "test-secret-at-least-32-chars-long-for-security"
	issuer := "myenglish-test"
	ttl := 15 * time.Minute

	manager := NewJWTManager(secret, issuer, ttl)

	raw, hash, err := manager.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	// Recompute hash from raw
	recomputedHash := HashToken(raw)
	if recomputedHash != hash {
		t.Errorf("hash mismatch: expected %s, got %s", hash, recomputedHash)
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	raw := "test-token-12345"

	hash1 := HashToken(raw)
	hash2 := HashToken(raw)

	if hash1 != hash2 {
		t.Errorf("hash is not deterministic: %s != %s", hash1, hash2)
	}

	// Different input should produce different hash
	differentRaw := "different-token-67890"
	hash3 := HashToken(differentRaw)
	if hash1 == hash3 {
		t.Error("different inputs produced same hash")
	}
}
