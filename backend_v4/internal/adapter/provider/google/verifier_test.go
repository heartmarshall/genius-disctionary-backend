package google

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVerifier_VerifyCode_Success(t *testing.T) {
	// Setup mock token server
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}

		// Validate form data
		if got := r.FormValue("grant_type"); got != "authorization_code" {
			t.Errorf("grant_type: got %q, want %q", got, "authorization_code")
		}
		if got := r.FormValue("code"); got != "test_code" {
			t.Errorf("code: got %q, want %q", got, "test_code")
		}
		if got := r.FormValue("client_id"); got != "test_client_id" {
			t.Errorf("client_id: got %q, want %q", got, "test_client_id")
		}
		if got := r.FormValue("client_secret"); got != "test_client_secret" {
			t.Errorf("client_secret: got %q, want %q", got, "test_client_secret")
		}
		if got := r.FormValue("redirect_uri"); got != "http://localhost:8080/callback" {
			t.Errorf("redirect_uri: got %q, want %q", got, "http://localhost:8080/callback")
		}

		// Return access token
		resp := tokenResponse{
			AccessToken: "test_access_token",
			IDToken:     "test_id_token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatal(err)
		}
	}))
	defer tokenSrv.Close()

	// Setup mock userinfo server
	userinfoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Validate Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test_access_token" {
			t.Errorf("Authorization: got %q, want %q", auth, "Bearer test_access_token")
		}

		// Return user info
		resp := userinfoResponse{
			ID:            "google_user_123",
			Email:         "user@example.com",
			VerifiedEmail: true,
			Name:          "Test User",
			Picture:       "https://example.com/avatar.jpg",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatal(err)
		}
	}))
	defer userinfoSrv.Close()

	// Override URLs to point to test servers
	origTokenURL := tokenURL
	origUserinfoURL := userinfoURL
	tokenURL = tokenSrv.URL
	userinfoURL = userinfoSrv.URL
	defer func() {
		tokenURL = origTokenURL
		userinfoURL = origUserinfoURL
	}()

	// Create verifier
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	verifier := NewVerifier("test_client_id", "test_client_secret", "http://localhost:8080/callback", logger)

	// Execute
	ctx := context.Background()
	identity, err := verifier.VerifyCode(ctx, "google", "test_code")

	// Assert
	if err != nil {
		t.Fatalf("VerifyCode() error = %v, want nil", err)
	}

	if identity.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", identity.Email, "user@example.com")
	}

	if identity.ProviderID != "google_user_123" {
		t.Errorf("ProviderID = %q, want %q", identity.ProviderID, "google_user_123")
	}

	if identity.Name == nil || *identity.Name != "Test User" {
		t.Errorf("Name = %v, want %q", identity.Name, "Test User")
	}

	if identity.AvatarURL == nil || *identity.AvatarURL != "https://example.com/avatar.jpg" {
		t.Errorf("AvatarURL = %v, want %q", identity.AvatarURL, "https://example.com/avatar.jpg")
	}
}

func TestVerifier_VerifyCode_UnverifiedEmail(t *testing.T) {

	// Setup mock servers
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := tokenResponse{
			AccessToken: "test_access_token",
			IDToken:     "test_id_token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer tokenSrv.Close()

	userinfoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := userinfoResponse{
			ID:            "google_user_123",
			Email:         "user@example.com",
			VerifiedEmail: false, // Unverified
			Name:          "Test User",
			Picture:       "https://example.com/avatar.jpg",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer userinfoSrv.Close()

	// Override URLs
	origTokenURL := tokenURL
	origUserinfoURL := userinfoURL
	tokenURL = tokenSrv.URL
	userinfoURL = userinfoSrv.URL
	defer func() {
		tokenURL = origTokenURL
		userinfoURL = origUserinfoURL
	}()

	// Create verifier
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	verifier := NewVerifier("test_client_id", "test_client_secret", "http://localhost:8080/callback", logger)

	// Execute
	ctx := context.Background()
	_, err := verifier.VerifyCode(ctx, "google", "test_code")

	// Assert
	if err == nil {
		t.Fatal("VerifyCode() error = nil, want error for unverified email")
	}

	expectedErr := "oauth: email not verified"
	if err.Error() != expectedErr {
		t.Errorf("error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestVerifier_VerifyCode_MissingName(t *testing.T) {

	// Setup mock servers
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := tokenResponse{
			AccessToken: "test_access_token",
			IDToken:     "test_id_token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer tokenSrv.Close()

	userinfoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := userinfoResponse{
			ID:            "google_user_123",
			Email:         "user@example.com",
			VerifiedEmail: true,
			Name:          "", // Missing name
			Picture:       "", // Missing picture
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer userinfoSrv.Close()

	// Override URLs
	origTokenURL := tokenURL
	origUserinfoURL := userinfoURL
	tokenURL = tokenSrv.URL
	userinfoURL = userinfoSrv.URL
	defer func() {
		tokenURL = origTokenURL
		userinfoURL = origUserinfoURL
	}()

	// Create verifier
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	verifier := NewVerifier("test_client_id", "test_client_secret", "http://localhost:8080/callback", logger)

	// Execute
	ctx := context.Background()
	identity, err := verifier.VerifyCode(ctx, "google", "test_code")

	// Assert
	if err != nil {
		t.Fatalf("VerifyCode() error = %v, want nil", err)
	}

	if identity.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", identity.Email, "user@example.com")
	}

	if identity.Name != nil {
		t.Errorf("Name = %v, want nil", identity.Name)
	}

	if identity.AvatarURL != nil {
		t.Errorf("AvatarURL = %v, want nil", identity.AvatarURL)
	}
}

func TestVerifier_VerifyCode_InvalidCode(t *testing.T) {

	// Setup mock server that returns 400
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		resp := errorResponse{
			Error:            "invalid_grant",
			ErrorDescription: "Invalid authorization code",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer tokenSrv.Close()

	// Override URLs
	origTokenURL := tokenURL
	tokenURL = tokenSrv.URL
	defer func() {
		tokenURL = origTokenURL
	}()

	// Create verifier
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	verifier := NewVerifier("test_client_id", "test_client_secret", "http://localhost:8080/callback", logger)

	// Execute
	ctx := context.Background()
	_, err := verifier.VerifyCode(ctx, "google", "invalid_code")

	// Assert
	if err == nil {
		t.Fatal("VerifyCode() error = nil, want error for invalid code")
	}

	expectedErr := "oauth: invalid or expired code"
	if err.Error() != expectedErr {
		t.Errorf("error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestVerifier_VerifyCode_Retry5xx(t *testing.T) {

	// Track number of calls
	var callCount int

	// Setup mock server that returns 500 first, then 200
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: return 500
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Second call: return success
		resp := tokenResponse{
			AccessToken: "test_access_token",
			IDToken:     "test_id_token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer tokenSrv.Close()

	userinfoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := userinfoResponse{
			ID:            "google_user_123",
			Email:         "user@example.com",
			VerifiedEmail: true,
			Name:          "Test User",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer userinfoSrv.Close()

	// Override URLs
	origTokenURL := tokenURL
	origUserinfoURL := userinfoURL
	tokenURL = tokenSrv.URL
	userinfoURL = userinfoSrv.URL
	defer func() {
		tokenURL = origTokenURL
		userinfoURL = origUserinfoURL
	}()

	// Create verifier
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	verifier := NewVerifier("test_client_id", "test_client_secret", "http://localhost:8080/callback", logger)

	// Execute
	ctx := context.Background()
	identity, err := verifier.VerifyCode(ctx, "google", "test_code")

	// Assert
	if err != nil {
		t.Fatalf("VerifyCode() error = %v, want nil (after retry)", err)
	}

	if callCount != 2 {
		t.Errorf("token server called %d times, want 2", callCount)
	}

	if identity.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", identity.Email, "user@example.com")
	}
}

func TestVerifier_VerifyCode_Retry5xxFails(t *testing.T) {

	// Track number of calls
	var callCount int

	// Setup mock server that always returns 500
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tokenSrv.Close()

	// Override URLs
	origTokenURL := tokenURL
	tokenURL = tokenSrv.URL
	defer func() {
		tokenURL = origTokenURL
	}()

	// Create verifier
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	verifier := NewVerifier("test_client_id", "test_client_secret", "http://localhost:8080/callback", logger)

	// Execute
	ctx := context.Background()
	_, err := verifier.VerifyCode(ctx, "google", "test_code")

	// Assert
	if err == nil {
		t.Fatal("VerifyCode() error = nil, want error after failed retry")
	}

	if callCount != 2 {
		t.Errorf("token server called %d times, want 2 (original + 1 retry)", callCount)
	}

	expectedErr := "oauth: google unavailable"
	if err.Error() != expectedErr {
		t.Errorf("error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestVerifier_VerifyCode_Timeout(t *testing.T) {

	// Setup mock server that delays longer than client timeout
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		resp := tokenResponse{
			AccessToken: "test_access_token",
			IDToken:     "test_id_token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer tokenSrv.Close()

	// Override URLs
	origTokenURL := tokenURL
	tokenURL = tokenSrv.URL
	defer func() {
		tokenURL = origTokenURL
	}()

	// Create verifier with very short timeout for testing
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	verifier := NewVerifier("test_client_id", "test_client_secret", "http://localhost:8080/callback", logger)
	verifier.httpClient.Timeout = 100 * time.Millisecond // Short timeout for test

	// Execute
	ctx := context.Background()
	_, err := verifier.VerifyCode(ctx, "google", "test_code")

	// Assert
	if err == nil {
		t.Fatal("VerifyCode() error = nil, want timeout error")
	}

	// Should contain "google unavailable" after timeout + retry
	expectedErr := "oauth: google unavailable"
	if err.Error() != expectedErr {
		t.Errorf("error = %q, want %q", err.Error(), expectedErr)
	}
}

// testWriter wraps testing.T to implement io.Writer for slog
type testWriter struct {
	t *testing.T
}

func (w testWriter) Write(p []byte) (n int, err error) {
	w.t.Log(string(p))
	return len(p), nil
}
