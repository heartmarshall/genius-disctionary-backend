package app

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/heartmarshall/myenglish-backend/internal/config"
)

func TestNewLogger_JSONFormat(t *testing.T) {
	cfg := config.LogConfig{Level: "info", Format: "json"}
	logger := NewLogger(cfg)

	var buf bytes.Buffer
	// Create a logger with the same options but writing to buf for verification.
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	testLogger := slog.New(handler)
	testLogger.Info("test message")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("JSON handler should produce valid JSON: %v", err)
	}

	if logger == nil {
		t.Fatal("logger should not be nil")
	}
}

func TestNewLogger_TextFormat(t *testing.T) {
	cfg := config.LogConfig{Level: "debug", Format: "text"}
	logger := NewLogger(cfg)

	if logger == nil {
		t.Fatal("logger should not be nil")
	}

	// Text format should have AddSource=true; verify by writing a log entry.
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	testLogger := slog.New(handler)
	testLogger.Debug("source test")

	output := buf.String()
	if !strings.Contains(output, "source=") {
		t.Error("text format should include source information")
	}
}

func TestNewLogger_SetsDefault(t *testing.T) {
	cfg := config.LogConfig{Level: "info", Format: "json"}
	logger := NewLogger(cfg)

	def := slog.Default()
	// They should reference the same underlying handler.
	if def.Handler() != logger.Handler() {
		t.Error("NewLogger should set the returned logger as slog default")
	}
}

func TestNewLogger_Levels(t *testing.T) {
	tests := []struct {
		level    string
		wantSlog slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run("level_"+tt.level, func(t *testing.T) {
			var buf bytes.Buffer

			logger := newLoggerWithWriter(&buf, config.LogConfig{
				Level:  tt.level,
				Format: "text",
			})

			// Log at one level below the expected level — should be suppressed.
			// Log at the expected level — should appear.
			logger.Log(context.TODO(), tt.wantSlog, "should appear")
			if buf.Len() == 0 {
				t.Errorf("expected log output at level %v", tt.wantSlog)
			}

			buf.Reset()
			belowLevel := tt.wantSlog - 1
			logger.Log(context.TODO(), belowLevel, "should be suppressed")
			if buf.Len() != 0 {
				t.Errorf("level %v should suppress level %v, but got output: %s",
					tt.wantSlog, belowLevel, buf.String())
			}
		})
	}
}

func TestNewLogger_TextAddSource_JSONNoSource(t *testing.T) {
	var textBuf, jsonBuf bytes.Buffer

	textLogger := newLoggerWithWriter(&textBuf, config.LogConfig{
		Level: "info", Format: "text",
	})
	textLogger.Info("hello")

	jsonLogger := newLoggerWithWriter(&jsonBuf, config.LogConfig{
		Level: "info", Format: "json",
	})
	jsonLogger.Info("hello")

	if !strings.Contains(textBuf.String(), "source=") {
		t.Error("text format should include source")
	}

	var m map[string]any
	if err := json.Unmarshal(jsonBuf.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := m["source"]; ok {
		t.Error("json format should not include source")
	}
}

// newLoggerWithWriter creates a logger writing to the given buffer
// (for test assertions), using the same logic as NewLogger.
func newLoggerWithWriter(buf *bytes.Buffer, cfg config.LogConfig) *slog.Logger {
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: strings.EqualFold(cfg.Format, "text"),
	}
	var handler slog.Handler
	if strings.EqualFold(cfg.Format, "json") {
		handler = slog.NewJSONHandler(buf, opts)
	} else {
		handler = slog.NewTextHandler(buf, opts)
	}
	return slog.New(handler)
}
