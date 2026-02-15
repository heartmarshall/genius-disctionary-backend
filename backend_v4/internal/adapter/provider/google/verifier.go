package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/auth"
)

var (
	// Made variables for testing purposes
	tokenURL    = "https://oauth2.googleapis.com/token"
	userinfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// Verifier exchanges Google OAuth authorization codes for user identity.
type Verifier struct {
	clientID     string
	clientSecret string
	redirectURI  string
	httpClient   *http.Client
	log          *slog.Logger
}

// NewVerifier creates a Google OAuth verifier.
// Parameters come from config.AuthConfig: GoogleClientID, GoogleClientSecret, GoogleRedirectURI.
func NewVerifier(clientID, clientSecret, redirectURI string, logger *slog.Logger) *Verifier {
	return &Verifier{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		log:          logger.With("adapter", "google_oauth"),
	}
}

// tokenResponse represents the response from Google's token endpoint.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// errorResponse represents Google's error response format.
type errorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// userinfoResponse represents the response from Google's userinfo endpoint.
type userinfoResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// VerifyCode exchanges an authorization code for user identity.
// The provider parameter is ignored (always "google"), but kept for interface compatibility.
func (v *Verifier) VerifyCode(ctx context.Context, provider, code string) (*auth.OAuthIdentity, error) {
	// Step 1: Exchange code for access token
	accessToken, err := v.exchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Step 2: Fetch user info
	userinfo, err := v.fetchUserinfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	// Step 3: Validate email is verified
	if !userinfo.VerifiedEmail {
		return nil, fmt.Errorf("oauth: email not verified")
	}

	// Step 4: Map to OAuthIdentity
	identity := &auth.OAuthIdentity{
		Email:      userinfo.Email,
		ProviderID: userinfo.ID,
	}

	// Optional fields: Name and Picture
	if userinfo.Name != "" {
		identity.Name = &userinfo.Name
	}
	if userinfo.Picture != "" {
		identity.AvatarURL = &userinfo.Picture
	}

	v.log.DebugContext(ctx, "google oauth success", slog.String("email", userinfo.Email))

	return identity, nil
}

// exchangeCode exchanges the authorization code for an access token.
func (v *Verifier) exchangeCode(ctx context.Context, code string) (string, error) {
	// Build form data
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", v.clientID)
	data.Set("client_secret", v.clientSecret)
	data.Set("redirect_uri", v.redirectURI)

	// Encode form data
	encodedData := data.Encode()

	// Create request with a reusable body (strings.Reader implements io.ReadSeeker)
	bodyReader := strings.NewReader(encodedData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Store GetBody function for retry (needed for HTTP client to replay body)
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(encodedData)), nil
	}

	// Execute with retry
	resp, err := v.doWithRetry(ctx, req)
	if err != nil {
		v.log.ErrorContext(ctx, "google oauth token exchange failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("oauth: google unavailable")
	}
	defer resp.Body.Close()

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		v.log.ErrorContext(ctx, "google oauth token exchange failed", slog.String("error", "failed to read response"))
		return "", fmt.Errorf("oauth: failed to read token response")
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errResp errorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			v.log.ErrorContext(ctx, "google oauth token exchange failed",
				slog.Int("status", resp.StatusCode),
				slog.String("error", errResp.Error))

			// 400 errors are typically invalid/expired codes
			if resp.StatusCode == http.StatusBadRequest {
				return "", fmt.Errorf("oauth: invalid or expired code")
			}
		}

		v.log.ErrorContext(ctx, "google oauth token exchange failed", slog.Int("status", resp.StatusCode))
		return "", fmt.Errorf("oauth: google unavailable")
	}

	// Parse success response
	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		v.log.ErrorContext(ctx, "google oauth token exchange failed", slog.String("error", "invalid json"))
		return "", fmt.Errorf("oauth: invalid token response")
	}

	if tokenResp.AccessToken == "" {
		v.log.ErrorContext(ctx, "google oauth token exchange failed", slog.String("error", "missing access_token"))
		return "", fmt.Errorf("oauth: invalid token response")
	}

	return tokenResp.AccessToken, nil
}

// fetchUserinfo fetches user information using the access token.
func (v *Verifier) fetchUserinfo(ctx context.Context, accessToken string) (*userinfoResponse, error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userinfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Execute with retry
	resp, err := v.doWithRetry(ctx, req)
	if err != nil {
		v.log.ErrorContext(ctx, "google oauth userinfo failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("oauth: failed to fetch user info")
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		v.log.ErrorContext(ctx, "google oauth userinfo failed", slog.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("oauth: failed to fetch user info")
	}

	// Parse response
	var userinfo userinfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userinfo); err != nil {
		v.log.ErrorContext(ctx, "google oauth userinfo failed", slog.String("error", "invalid json"))
		return nil, fmt.Errorf("oauth: invalid userinfo response")
	}

	// Validate required fields
	if userinfo.ID == "" || userinfo.Email == "" {
		v.log.ErrorContext(ctx, "google oauth userinfo failed", slog.String("error", "missing required fields"))
		return nil, fmt.Errorf("oauth: invalid userinfo response")
	}

	return &userinfo, nil
}

// doWithRetry executes an HTTP request with retry logic.
// Retries once on 5xx errors or network errors with 500ms backoff.
// Note: For POST requests, the body must be reusable (e.g., strings.Reader).
func (v *Verifier) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Check context before first attempt
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// First attempt
	resp, err := v.httpClient.Do(req)
	if err != nil || (resp != nil && resp.StatusCode >= 500) {
		// Close response if we got one
		if resp != nil {
			resp.Body.Close()
		}

		// Check context before retry
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Wait before retry (context-aware)
		select {
		case <-time.After(500 * time.Millisecond):
			// Continue to retry
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Retry - Note: req.Body needs to be seekable for POST to work
		// Since we use strings.Reader in exchangeCode, this is safe
		resp, err = v.httpClient.Do(req)
	}

	return resp, err
}
