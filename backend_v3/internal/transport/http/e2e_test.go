package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/heartmarshall/my-english/internal/config"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository"
	"github.com/heartmarshall/my-english/internal/service"
	transportHttp "github.com/heartmarshall/my-english/internal/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// testApp represents a test application instance with all dependencies.
type testApp struct {
	pool      *pgxpool.Pool
	repos     *repository.Registry
	services  *service.Services
	handler   http.Handler
	logger    *slog.Logger
	container testcontainers.Container
}

// setupTestApp creates a test application with a PostgreSQL container.
func setupTestApp(t *testing.T) *testApp {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test_db",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort("5432/tcp"),
		).WithDeadline(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get container connection details
	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Build DSN
	dsn := fmt.Sprintf("postgres://test:test@%s:%s/test_db?sslmode=disable", host, port.Port())

	// Wait for database to be ready
	var pool *pgxpool.Pool
	for i := 0; i < 10; i++ {
		pool, err = pgxpool.New(ctx, dsn)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				break
			}
			pool.Close()
		}
		time.Sleep(time.Second)
	}
	require.NoError(t, err, "Failed to connect to test database")

	// Run migrations
	err = runMigrations(ctx, pool, dsn)
	require.NoError(t, err, "Failed to run migrations")

	// Initialize repositories
	repos := repository.NewRegistry(pool)
	txManager := database.NewTxManager(pool)

	// Initialize services
	services, err := service.NewServices(service.Deps{
		Repos:     repos,
		TxManager: txManager,
		Providers: nil, // No external providers for e2e tests
	})
	require.NoError(t, err, "Failed to initialize services")

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Only log errors in tests
	}))

	// Create test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:           "localhost",
			Port:           8080,
			ReadTimeout:    15 * time.Second,
			WriteTimeout:   15 * time.Second,
			RequestTimeout: 30 * time.Second,
		},
		Database: config.DatabaseConfig{
			Host:            host,
			Port:            port.Int(),
			User:            "test",
			Password:        "test",
			Database:        "test_db",
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		GraphQL: config.GraphQLConfig{
			EnablePlayground:    false,
			EnableIntrospection: true,
			QueryCacheSize:      1000,
		},
		Log: config.LogConfig{
			Level:  "error",
			Format: "text",
		},
	}

	// Create HTTP handler
	handler := transportHttp.NewHandler(cfg, logger, services, repos)

	return &testApp{
		pool:      pool,
		repos:     repos,
		services:  services,
		handler:   handler,
		logger:    logger,
		container: container,
	}
}

// teardownTestApp cleans up test resources.
func (app *testApp) teardown(t *testing.T) {
	t.Helper()
	if app.pool != nil {
		app.pool.Close()
	}
	if app.container != nil {
		ctx := context.Background()
		err := app.container.Terminate(ctx)
		require.NoError(t, err, "Failed to terminate container")
	}
}

// runMigrations executes database migrations using goose.
func runMigrations(ctx context.Context, pool *pgxpool.Pool, dsn string) error {
	// Find migrations directory
	migrationsPath := filepath.Join("..", "..", "..", "migrations")
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Read migration files
	files, err := os.ReadDir(absPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name (they should be prefixed with timestamps)
	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
			migrationFiles = append(migrationFiles, filepath.Join(absPath, file.Name()))
		}
	}

	// Execute migrations in order
	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		// Extract "Up" section from goose migration
		sql := extractGooseUp(string(content))
		if sql == "" {
			continue // Skip if no Up section
		}

		// Execute migration
		if _, err := pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	return nil
}

// extractGooseUp extracts the SQL from the "-- +goose Up" section.
func extractGooseUp(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inUpSection := false

	for _, line := range lines {
		if strings.Contains(line, "-- +goose Down") {
			break
		}
		if strings.Contains(line, "-- +goose Up") {
			inUpSection = true
			continue
		}
		if inUpSection {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// graphQLRequest represents a GraphQL request.
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// graphQLResponse represents a GraphQL response.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors,omitempty"`
}

// graphQLError represents a GraphQL error.
type graphQLError struct {
	Message    string                 `json:"message"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// executeGraphQL executes a GraphQL request and returns the response.
func (app *testApp) executeGraphQL(t *testing.T, query string, variables map[string]interface{}) *graphQLResponse {
	t.Helper()

	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-request-id")

	rec := httptest.NewRecorder()
	app.handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())

	var resp graphQLResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err, "Failed to unmarshal response: %s", rec.Body.String())

	return &resp
}

// executeGraphQLWithError executes a GraphQL request and expects an error.
func (app *testApp) executeGraphQLWithError(t *testing.T, query string, variables map[string]interface{}) *graphQLResponse {
	t.Helper()

	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	app.handler.ServeHTTP(rec, req)

	var resp graphQLResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	require.NotEmpty(t, resp.Errors, "Expected GraphQL errors but got none")

	return &resp
}

// readJSONFile reads a JSON file and unmarshals it.
func readJSONFile(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	err = json.Unmarshal(data, v)
	require.NoError(t, err)
}

// prettyJSON formats JSON for debugging.
func prettyJSON(t *testing.T, data interface{}) string {
	t.Helper()
	b, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	return string(b)
}

// extractString extracts a string value from JSON by path.
func extractString(t *testing.T, data json.RawMessage, path ...string) string {
	t.Helper()
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	require.NoError(t, err)

	var current interface{} = m
	for i, p := range path {
		mm, ok := current.(map[string]interface{})
		require.True(t, ok, "Path element %d (%s) is not a map", i, p)
		current, ok = mm[p]
		require.True(t, ok, "Path element %s not found", p)
	}

	str, ok := current.(string)
	require.True(t, ok, "Value at path %v is not a string", path)
	return str
}

// extractInt extracts an int value from JSON by path.
func extractInt(t *testing.T, data json.RawMessage, path ...string) int {
	t.Helper()
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	require.NoError(t, err)

	var current interface{} = m
	for i, p := range path {
		mm, ok := current.(map[string]interface{})
		require.True(t, ok, "Path element %d (%s) is not a map", i, p)
		current, ok = mm[p]
		require.True(t, ok, "Path element %s not found", p)
	}

	// Handle both int and float64 (JSON numbers)
	var num float64
	switch v := current.(type) {
	case float64:
		num = v
	case int:
		num = float64(v)
	default:
		require.Fail(t, "Value at path %v is not a number", path)
	}

	return int(num)
}

// extractArray extracts an array from JSON by path.
func extractArray(t *testing.T, data json.RawMessage, path ...string) []interface{} {
	t.Helper()
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	require.NoError(t, err)

	var current interface{} = m
	for i, p := range path {
		mm, ok := current.(map[string]interface{})
		require.True(t, ok, "Path element %d (%s) is not a map", i, p)
		current, ok = mm[p]
		require.True(t, ok, "Path element %s not found", p)
	}

	arr, ok := current.([]interface{})
	require.True(t, ok, "Value at path %v is not an array", path)
	return arr
}

// extractBool extracts a bool value from JSON by path.
func extractBool(t *testing.T, data json.RawMessage, path ...string) bool {
	t.Helper()
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	require.NoError(t, err)

	var current interface{} = m
	for i, p := range path {
		mm, ok := current.(map[string]interface{})
		require.True(t, ok, "Path element %d (%s) is not a map", i, p)
		current, ok = mm[p]
		require.True(t, ok, "Path element %s not found", p)
	}

	b, ok := current.(bool)
	require.True(t, ok, "Value at path %v is not a bool", path)
	return b
}

// extractObject extracts an object (map) from JSON by path.
func extractObject(t *testing.T, data json.RawMessage, path ...string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	require.NoError(t, err)

	var current interface{} = m
	for i, p := range path {
		mm, ok := current.(map[string]interface{})
		require.True(t, ok, "Path element %d (%s) is not a map", i, p)
		current, ok = mm[p]
		require.True(t, ok, "Path element %s not found", p)
	}

	obj, ok := current.(map[string]interface{})
	require.True(t, ok, "Value at path %v is not an object", path)
	return obj
}

// readBody reads the response body.
func readBody(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
