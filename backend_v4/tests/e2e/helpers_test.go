//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/audit"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/card"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/entry"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/example"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/image"
	inboxrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/inbox"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/pronunciation"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/reviewlog"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/sense"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/session"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/token"
	topicrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/topic"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/translation"
	userrepo "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/user"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/provider/freedict"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/provider/translate"
	authpkg "github.com/heartmarshall/myenglish-backend/internal/auth"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	authsvc "github.com/heartmarshall/myenglish-backend/internal/service/auth"
	"github.com/heartmarshall/myenglish-backend/internal/service/content"
	"github.com/heartmarshall/myenglish-backend/internal/service/dictionary"
	inboxsvc "github.com/heartmarshall/myenglish-backend/internal/service/inbox"
	"github.com/heartmarshall/myenglish-backend/internal/service/refcatalog"
	"github.com/heartmarshall/myenglish-backend/internal/service/study"
	topicsvc "github.com/heartmarshall/myenglish-backend/internal/service/topic"
	usersvc "github.com/heartmarshall/myenglish-backend/internal/service/user"
	gqlpkg "github.com/heartmarshall/myenglish-backend/internal/transport/graphql"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/dataloader"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/generated"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/resolver"
	"github.com/heartmarshall/myenglish-backend/internal/transport/middleware"
	"github.com/heartmarshall/myenglish-backend/internal/transport/rest"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// GraphQL assertion / extraction helpers.
// ---------------------------------------------------------------------------

// gqlData extracts the "data" map from a GraphQL response.
func gqlData(t *testing.T, result map[string]any) map[string]any {
	t.Helper()
	data, ok := result["data"].(map[string]any)
	require.True(t, ok, "expected data object in response")
	return data
}

// gqlPayload extracts a specific field from the data map.
func gqlPayload(t *testing.T, result map[string]any, field string) map[string]any {
	t.Helper()
	data := gqlData(t, result)
	payload, ok := data[field].(map[string]any)
	require.True(t, ok, "expected %q in data", field)
	return payload
}

// gqlErrorCode extracts the error code from the first GraphQL error.
func gqlErrorCode(t *testing.T, result map[string]any) string {
	t.Helper()
	errors, ok := result["errors"].([]any)
	require.True(t, ok, "expected errors array")
	require.NotEmpty(t, errors)

	firstErr, ok := errors[0].(map[string]any)
	require.True(t, ok)
	extensions, ok := firstErr["extensions"].(map[string]any)
	require.True(t, ok, "expected extensions in error")

	code, ok := extensions["code"].(string)
	require.True(t, ok, "expected code string in extensions")
	return code
}

// requireNoErrors asserts that the GraphQL response has no errors.
func requireNoErrors(t *testing.T, result map[string]any) {
	t.Helper()
	if errs, ok := result["errors"]; ok && errs != nil {
		t.Fatalf("unexpected GraphQL errors: %v", errs)
	}
}

// ---------------------------------------------------------------------------
// testServer wraps the full-stack HTTP server for E2E tests.
// ---------------------------------------------------------------------------

type testServer struct {
	URL    string
	Client *http.Client
	Pool   *pgxpool.Pool
	jwt    *authpkg.JWTManager
}

// testLogWriter adapts testing.T to io.Writer for slog.
type testLogWriter struct{ t *testing.T }

func (w testLogWriter) Write(p []byte) (int, error) {
	w.t.Helper()
	w.t.Log(string(p))
	return len(p), nil
}

// ---------------------------------------------------------------------------
// Mock OAuth verifier (not used in E2E tests)
// ---------------------------------------------------------------------------

type mockOAuthVerifier struct{}

func (m *mockOAuthVerifier) VerifyCode(_ context.Context, _, _ string) (*authpkg.OAuthIdentity, error) {
	return nil, fmt.Errorf("mock: oauth not supported in tests")
}

// ---------------------------------------------------------------------------
// setupTestServer bootstraps the full application stack backed by
// a real PostgreSQL container (shared via testhelper).
// ---------------------------------------------------------------------------

func setupTestServer(t *testing.T) *testServer {
	t.Helper()

	// 1. Get pool from testcontainers-backed helper.
	pool := testhelper.SetupTestDB(t)

	// 2. Infrastructure.
	logger := slog.New(slog.NewTextHandler(testLogWriter{t}, nil))
	txm := postgres.NewTxManager(pool)

	// 3. Repositories.
	auditRepo := audit.New(pool)
	cardRepo := card.New(pool)
	entryRepo := entry.New(pool)
	exampleRepo := example.New(pool, txm)
	imageRepo := image.New(pool)
	inboxRepo := inboxrepo.New(pool)
	pronunciationRepo := pronunciation.New(pool)
	refentryRepo := refentry.New(pool, txm)
	reviewlogRepo := reviewlog.New(pool)
	senseRepo := sense.New(pool, txm)
	sessionRepo := session.New(pool)
	tokenRepo := token.New(pool)
	topicRepo := topicrepo.New(pool)
	translationRepo := translation.New(pool, txm)
	userRepo := userrepo.New(pool)

	// 4. External providers.
	dictProvider := freedict.NewProvider(logger)
	transProvider := translate.NewStub()

	// 5. JWT manager with a test secret (>= 32 chars).
	jwtSecret := "test-secret-at-least-32-chars-long!!"
	jwtIssuer := "test-issuer"
	accessTTL := 15 * time.Minute
	jwtMgr := authpkg.NewJWTManager(jwtSecret, jwtIssuer, accessTTL)

	// 6. Mock OAuth verifier — never actually called.
	oauthVerifier := &mockOAuthVerifier{}

	// 7. Services.
	authService := authsvc.NewService(
		logger, userRepo, userRepo, tokenRepo, txm, oauthVerifier, jwtMgr,
		config.AuthConfig{
			JWTSecret:       jwtSecret,
			JWTIssuer:       jwtIssuer,
			AccessTokenTTL:  accessTTL,
			RefreshTokenTTL: 720 * time.Hour,
		},
	)

	userService := usersvc.NewService(logger, userRepo, userRepo, auditRepo, txm)

	refCatalogService := refcatalog.NewService(logger, refentryRepo, txm, dictProvider, transProvider)

	srsConfig := domain.SRSConfig{
		DefaultEaseFactor:    2.5,
		MinEaseFactor:        1.3,
		MaxIntervalDays:      365,
		GraduatingInterval:   1,
		LearningSteps:        []time.Duration{time.Minute, 10 * time.Minute},
		NewCardsPerDay:       20,
		ReviewsPerDay:        200,
		EasyInterval:         4,
		RelearningSteps:      []time.Duration{10 * time.Minute},
		IntervalModifier:     1.0,
		HardIntervalModifier: 1.2,
		EasyBonus:            1.3,
		LapseNewInterval:     0.0,
		UndoWindowMinutes:    10,
	}

	dictionaryService := dictionary.NewService(
		logger, entryRepo, senseRepo, translationRepo, exampleRepo,
		pronunciationRepo, imageRepo, cardRepo, auditRepo, txm,
		refCatalogService, config.DictionaryConfig{
			MaxEntriesPerUser: 10000,
			DefaultEaseFactor: 2.5,
		},
	)

	contentService := content.NewService(
		logger, entryRepo, senseRepo, translationRepo, exampleRepo,
		imageRepo, auditRepo, txm,
	)

	studyService := study.NewService(
		logger, cardRepo, reviewlogRepo, sessionRepo, entryRepo,
		senseRepo, userRepo, auditRepo, txm, srsConfig,
	)

	topicService := topicsvc.NewService(logger, topicRepo, entryRepo, auditRepo, txm)

	inboxService := inboxsvc.NewService(logger, inboxRepo)

	// 8. GraphQL resolver + handler.
	res := resolver.NewResolver(
		logger, dictionaryService, contentService, studyService,
		topicService, inboxService, userService,
	)

	schema := generated.NewExecutableSchema(generated.Config{Resolvers: res})
	gqlSrv := gqlhandler.NewDefaultServer(schema)
	gqlSrv.SetErrorPresenter(gqlpkg.NewErrorPresenter(logger))

	// 9. DataLoader repositories.
	dlRepos := &dataloader.Repos{
		Sense:         senseRepo,
		Translation:   translationRepo,
		Example:       exampleRepo,
		Pronunciation: pronunciationRepo,
		Image:         imageRepo,
		Card:          cardRepo,
		Topic:         topicRepo,
		ReviewLog:     reviewlogRepo,
	}

	// 10. Middleware chain.
	graphqlHandler := middleware.Chain(
		middleware.Recovery(logger),
		middleware.RequestID(),
		middleware.CORS(config.CORSConfig{
			AllowedOrigins:   "*",
			AllowedMethods:   "GET,POST,OPTIONS",
			AllowedHeaders:   "Authorization,Content-Type",
			AllowCredentials: true,
			MaxAge:           86400,
		}),
		middleware.Auth(authService),
		middleware.Middleware(dataloader.Middleware(dlRepos)),
	)(gqlSrv)

	// 11. Mux.
	mux := http.NewServeMux()

	healthHandler := rest.NewHealthHandler(pool, "test-version")
	mux.HandleFunc("GET /live", healthHandler.Live)
	mux.HandleFunc("GET /ready", healthHandler.Ready)
	mux.HandleFunc("GET /health", healthHandler.Health)
	mux.Handle("POST /query", graphqlHandler)
	mux.Handle("OPTIONS /query", graphqlHandler)

	// 12. httptest server.
	srv := httptest.NewServer(mux)
	t.Cleanup(func() { srv.Close() })

	return &testServer{
		URL:    srv.URL,
		Client: srv.Client(),
		Pool:   pool,
		jwt:    jwtMgr,
	}
}

// ---------------------------------------------------------------------------
// graphqlQuery sends a GraphQL POST request and returns status + decoded body.
// ---------------------------------------------------------------------------

func (ts *testServer) graphqlQuery(t *testing.T, query string, variables map[string]any, token string) (int, map[string]any) {
	t.Helper()

	body := map[string]any{
		"query":     query,
		"variables": variables,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal graphql body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/query", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := ts.Client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.StatusCode, result
}

// ---------------------------------------------------------------------------
// createTestUserAndGetToken inserts a user + settings directly into the DB
// and returns a valid JWT access token for that user.
// ---------------------------------------------------------------------------

func createTestUserAndGetToken(t *testing.T, ts *testServer) string {
	t.Helper()

	userID := uuid.New()
	now := time.Now()

	// Insert user.
	_, err := ts.Pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, name, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID,
		fmt.Sprintf("test-%s@example.com", userID.String()[:8]),
		fmt.Sprintf("test-%s", userID.String()[:8]),
		"Test User",
		now, now,
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	// Insert default user settings.
	_, err = ts.Pool.Exec(context.Background(),
		`INSERT INTO user_settings (user_id, new_cards_per_day, reviews_per_day, max_interval_days, timezone, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, 20, 200, 365, "UTC", now,
	)
	if err != nil {
		t.Fatalf("insert test settings: %v", err)
	}

	// Generate JWT.
	tok, err := ts.jwt.GenerateAccessToken(userID)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	return tok
}

// ---------------------------------------------------------------------------
// createTestUserWithID is like createTestUserAndGetToken but also returns
// the user's UUID (needed for DB verification and seed helpers).
// ---------------------------------------------------------------------------

func createTestUserWithID(t *testing.T, ts *testServer) (string, uuid.UUID) {
	t.Helper()

	userID := uuid.New()
	now := time.Now()

	_, err := ts.Pool.Exec(context.Background(),
		`INSERT INTO users (id, email, username, name, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID,
		fmt.Sprintf("test-%s@example.com", userID.String()[:8]),
		fmt.Sprintf("test-%s", userID.String()[:8]),
		"Test User",
		now, now,
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	_, err = ts.Pool.Exec(context.Background(),
		`INSERT INTO user_settings (user_id, new_cards_per_day, reviews_per_day, max_interval_days, timezone, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, 20, 200, 365, "UTC", now,
	)
	if err != nil {
		t.Fatalf("insert test settings: %v", err)
	}

	tok, err := ts.jwt.GenerateAccessToken(userID)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	return tok, userID
}
