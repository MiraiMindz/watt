package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yourusername/bolt/core"
)

// TestLogger tests basic request logging.
func TestLogger(t *testing.T) {
	var logBuf bytes.Buffer

	config := LoggerConfig{
		Output: &logBuf,
		Format: "json",
	}

	middleware := LoggerWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Create context with request info
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/users")

	// Execute handler
	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check log output
	if logBuf.Len() == 0 {
		t.Fatal("expected log output, got none")
	}

	// Parse JSON log
	var logEntry LogEntry
	if err := json.Unmarshal(logBuf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}

	// Verify log fields
	if logEntry.Method != "GET" {
		t.Errorf("expected method GET, got %s", logEntry.Method)
	}

	if logEntry.Path != "/users" {
		t.Errorf("expected path /users, got %s", logEntry.Path)
	}

	if logEntry.Status != 200 {
		t.Errorf("expected status 200, got %d", logEntry.Status)
	}

	if logEntry.DurationMS < 0 {
		t.Errorf("expected positive duration, got %f", logEntry.DurationMS)
	}
}

// TestLoggerSkipPaths tests path skipping.
func TestLoggerSkipPaths(t *testing.T) {
	var logBuf bytes.Buffer

	config := LoggerConfig{
		Output:    &logBuf,
		Format:    "json",
		SkipPaths: []string{"/health", "/metrics"},
	}

	middleware := LoggerWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	// Test skipped path
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/health")

	_ = handler(ctx)

	// Should not log
	if logBuf.Len() > 0 {
		t.Error("expected no log output for skipped path")
	}

	// Test non-skipped path
	logBuf.Reset()
	ctx.SetPath("/users")

	_ = handler(ctx)

	// Should log
	if logBuf.Len() == 0 {
		t.Error("expected log output for non-skipped path")
	}
}

// TestLoggerWithError tests logging when handler returns error.
func TestLoggerWithError(t *testing.T) {
	var logBuf bytes.Buffer

	config := LoggerConfig{
		Output: &logBuf,
		Format: "json",
	}

	middleware := LoggerWithConfig(config)

	expectedErr := errors.New("handler error")
	handler := middleware(func(c *core.Context) error {
		return expectedErr
	})

	ctx := &core.Context{}
	ctx.SetMethod("POST")
	ctx.SetPath("/users")

	err := handler(ctx)

	if err != expectedErr {
		t.Errorf("expected error to be passed through, got %v", err)
	}

	// Check log output includes error
	if logBuf.Len() == 0 {
		t.Fatal("expected log output, got none")
	}

	var logEntry LogEntry
	if err := json.Unmarshal(logBuf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}

	if logEntry.Error == "" {
		t.Error("expected error in log entry")
	}
	if logEntry.Error != "handler error" {
		t.Errorf("expected error 'handler error', got '%s'", logEntry.Error)
	}
}

// TestLoggerTextFormat tests text format logging.
func TestLoggerTextFormat(t *testing.T) {
	var logBuf bytes.Buffer

	config := LoggerConfig{
		Output: &logBuf,
		Format: "text",
	}

	middleware := LoggerWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api/users")

	_ = handler(ctx)

	// Check log output (should be text, not JSON)
	logOutput := logBuf.String()
	if logOutput == "" {
		t.Fatal("expected log output, got none")
	}

	// Should contain method and path
	if !strings.Contains(logOutput, "GET") {
		t.Error("expected GET in log output")
	}

	if !strings.Contains(logOutput, "/api/users") {
		t.Error("expected path in log output")
	}
}

// TestDefaultLoggerConfig tests default configuration.
func TestDefaultLoggerConfig(t *testing.T) {
	config := DefaultLoggerConfig()

	if config.Output == nil {
		t.Error("expected Output to be set")
	}

	if config.Format != "json" {
		t.Errorf("expected format json, got %s", config.Format)
	}

	if len(config.SkipPaths) != 0 {
		t.Error("expected empty SkipPaths by default")
	}
}

// TestLoggerDefaultStatus tests that status defaults to 200.
func TestLoggerDefaultStatus(t *testing.T) {
	var logBuf bytes.Buffer

	config := LoggerConfig{
		Output: &logBuf,
		Format: "json",
	}

	middleware := LoggerWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		// Don't set status explicitly
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	_ = handler(ctx)

	var logEntry LogEntry
	if err := json.Unmarshal(logBuf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}

	// Should default to 200
	if logEntry.Status != 200 {
		t.Errorf("expected default status 200, got %d", logEntry.Status)
	}
}

// BenchmarkLogger benchmarks logger middleware overhead.
func BenchmarkLogger(b *testing.B) {
	var logBuf bytes.Buffer

	config := LoggerConfig{
		Output: &logBuf,
		Format: "json",
	}

	middleware := LoggerWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logBuf.Reset()
		_ = handler(ctx)
	}
}

// BenchmarkLoggerSkipPath benchmarks skipped path performance.
func BenchmarkLoggerSkipPath(b *testing.B) {
	var logBuf bytes.Buffer

	config := LoggerConfig{
		Output:    &logBuf,
		Format:    "json",
		SkipPaths: []string{"/health"},
	}

	middleware := LoggerWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/health") // Skipped path

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}

// TestLoggerFunction tests Logger() wrapper function.
func TestLoggerFunction(t *testing.T) {
	// This test ensures Logger() function is called and uses defaults
	middleware := Logger()

	if middleware == nil {
		t.Fatal("Logger() returned nil")
	}

	// Verify it works by executing it
	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// errorWriter always returns an error on Write.
type errorWriter struct{}

func (ew *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write error")
}

// TestLogJSONError tests logJSON error handling.
func TestLogJSONError(t *testing.T) {
	// Create an error writer that fails on write
	ew := &errorWriter{}

	config := LoggerConfig{
		Output: ew,
		Format: "json",
	}

	middleware := LoggerWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	// Should not panic even if logging fails
	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestLogTextError tests logText error handling.
func TestLogTextError(t *testing.T) {
	// Create an error writer that fails on write
	ew := &errorWriter{}

	config := LoggerConfig{
		Output: ew,
		Format: "text",
	}

	middleware := LoggerWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	// Should not panic even if logging fails
	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestLogTextErrorWithHandlerError tests logText with both write and handler errors.
func TestLogTextErrorWithHandlerError(t *testing.T) {
	ew := &errorWriter{}

	config := LoggerConfig{
		Output: ew,
		Format: "text",
	}

	middleware := LoggerWithConfig(config)

	expectedErr := errors.New("handler error")
	handler := middleware(func(c *core.Context) error {
		return expectedErr
	})

	ctx := &core.Context{}
	ctx.SetMethod("POST")
	ctx.SetPath("/test")

	// Should return handler error even if logging fails
	err := handler(ctx)
	if err != expectedErr {
		t.Errorf("expected handler error, got %v", err)
	}
}

// TestLoggerWithNilOutput tests logger with nil output using defaults.
func TestLoggerWithNilOutput(t *testing.T) {
	config := LoggerConfig{
		Output: nil, // Should use os.Stdout
		Format: "json",
	}

	middleware := LoggerWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestLoggerWithEmptyFormat tests logger with empty format (should use default).
func TestLoggerWithEmptyFormat(t *testing.T) {
	var logBuf bytes.Buffer

	config := LoggerConfig{
		Output: &logBuf,
		Format: "", // Should default to "json"
	}

	middleware := LoggerWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	_ = handler(ctx)

	// Should produce JSON output (default)
	if logBuf.Len() == 0 {
		t.Error("expected log output")
	}
}
