package middleware

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/yourusername/bolt/core"
)

// TestRecovery tests basic panic recovery.
func TestRecovery(t *testing.T) {
	middleware := Recovery()

	// Create handler that panics
	handler := middleware(func(c *core.Context) error {
		panic("test panic")
	})

	// Execute handler - should NOT panic (that's the point of recovery)
	ctx := &core.Context{}

	// This should complete without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic was not recovered: %v", r)
		}
	}()

	_ = handler(ctx)

	// Check that response was written (500 error)
	if !ctx.Written() {
		t.Error("expected response to be written after panic recovery")
	}

	if ctx.StatusCode() != 500 {
		t.Errorf("expected status 500 after panic, got %d", ctx.StatusCode())
	}
}

// TestRecoveryNoPanic tests that recovery doesn't interfere with normal execution.
func TestRecoveryNoPanic(t *testing.T) {
	middleware := Recovery()

	called := false
	handler := middleware(func(c *core.Context) error {
		called = true
		return nil
	})

	ctx := &core.Context{}
	err := handler(ctx)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler was not called")
	}
}

// TestRecoveryWithError tests recovery when handler returns error.
func TestRecoveryWithError(t *testing.T) {
	middleware := Recovery()

	expectedErr := errors.New("handler error")
	handler := middleware(func(c *core.Context) error {
		return expectedErr
	})

	ctx := &core.Context{}
	err := handler(ctx)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// TestRecoveryWithConfig tests configurable recovery.
func TestRecoveryWithConfig(t *testing.T) {
	var logBuf bytes.Buffer

	config := RecoveryConfig{
		PrintStack: true,
		StackSize:  4 << 10,
		LogOutput:  &logBuf,
		Handler: func(c *core.Context, err interface{}) error {
			return c.JSON(500, map[string]string{
				"error": "custom panic handler",
			})
		},
	}

	middleware := RecoveryWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		panic("configured panic")
	})

	ctx := &core.Context{}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic was not recovered: %v", r)
		}
	}()

	_ = handler(ctx)

	// Check that response was written
	if !ctx.Written() {
		t.Error("expected response to be written after panic recovery")
	}

	// Check that log was written
	if logBuf.Len() == 0 {
		t.Error("expected log output, got none")
	}

	// Check that panic message is in log
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "configured panic") {
		t.Error("expected panic message in log output")
	}

	// Check that stack trace is in log
	if !strings.Contains(logOutput, "goroutine") {
		t.Error("expected stack trace in log output")
	}
}

// TestRecoveryWithConfigNoPrintStack tests recovery without stack printing.
func TestRecoveryWithConfigNoPrintStack(t *testing.T) {
	var logBuf bytes.Buffer

	config := RecoveryConfig{
		PrintStack: false,
		LogOutput:  &logBuf,
	}

	middleware := RecoveryWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		panic("no stack panic")
	})

	ctx := &core.Context{}
	_ = handler(ctx)

	// Log should be empty (no stack printed)
	if logBuf.Len() > 0 {
		t.Error("expected no log output when PrintStack=false")
	}
}

// TestDefaultRecoveryConfig tests default config.
func TestDefaultRecoveryConfig(t *testing.T) {
	config := DefaultRecoveryConfig()

	if !config.PrintStack {
		t.Error("expected PrintStack to be true by default")
	}

	if config.StackSize != 4<<10 {
		t.Errorf("expected StackSize 4KB, got %d", config.StackSize)
	}

	if config.LogOutput != nil {
		t.Error("expected LogOutput to be nil (use default)")
	}

	if config.Handler != nil {
		t.Error("expected Handler to be nil (use default)")
	}
}

// BenchmarkRecovery benchmarks recovery middleware overhead.
func BenchmarkRecovery(b *testing.B) {
	middleware := Recovery()

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}

// BenchmarkRecoveryWithPanic benchmarks panic recovery performance.
func BenchmarkRecoveryWithPanic(b *testing.B) {
	middleware := Recovery()

	handler := middleware(func(c *core.Context) error {
		panic("benchmark panic")
	})

	ctx := &core.Context{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}

// TestRecoveryWithStringPanic tests recovery from string panic.
func TestRecoveryWithStringPanic(t *testing.T) {
	var logBuf bytes.Buffer

	config := RecoveryConfig{
		PrintStack: true,
		LogOutput:  &logBuf,
	}

	middleware := RecoveryWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		panic("string panic")
	})

	ctx := &core.Context{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic was not recovered: %v", r)
		}
	}()

	_ = handler(ctx)

	// Check logs contain panic info
	if logBuf.Len() == 0 {
		t.Error("expected log output")
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "string panic") {
		t.Error("expected panic message in log")
	}
}

// TestRecoveryWithNilHandler tests recovery with nil custom handler.
func TestRecoveryWithNilHandler(t *testing.T) {
	config := RecoveryConfig{
		PrintStack: false,
		Handler:    nil, // Use default handler
	}

	middleware := RecoveryWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		panic("nil handler panic")
	})

	ctx := &core.Context{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic was not recovered: %v", r)
		}
	}()

	_ = handler(ctx)

	if ctx.StatusCode() != 500 {
		t.Errorf("expected status 500, got %d", ctx.StatusCode())
	}
}
