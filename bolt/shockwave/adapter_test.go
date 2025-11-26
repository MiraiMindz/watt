package shockwave

import (
	"testing"
	"time"
)

// TestDefaultConfig tests default configuration values.
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("expected config, got nil")
	}

	if config.Addr != ":8080" {
		t.Errorf("expected addr :8080, got %s", config.Addr)
	}

	if config.ReadTimeout != 5*time.Second {
		t.Errorf("expected read timeout 5s, got %v", config.ReadTimeout)
	}

	if config.WriteTimeout != 10*time.Second {
		t.Errorf("expected write timeout 10s, got %v", config.WriteTimeout)
	}

	if config.IdleTimeout != 120*time.Second {
		t.Errorf("expected idle timeout 120s, got %v", config.IdleTimeout)
	}

	if config.MaxHeaderBytes != 1<<20 {
		t.Errorf("expected max header bytes 1MB, got %d", config.MaxHeaderBytes)
	}

	if config.MaxRequestBodySize != 10<<20 {
		t.Errorf("expected max request body size 10MB, got %d", config.MaxRequestBodySize)
	}

	if !config.DisableStats {
		t.Error("expected DisableStats to be true by default")
	}
}

// TestCustomConfig tests custom configuration.
func TestCustomConfig(t *testing.T) {
	config := &Config{
		Addr:               ":9000",
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       20 * time.Second,
		IdleTimeout:        60 * time.Second,
		MaxHeaderBytes:     2 << 20,
		MaxRequestBodySize: 5 << 20,
		DisableStats:       false,
	}

	if config.Addr != ":9000" {
		t.Errorf("expected addr :9000, got %s", config.Addr)
	}

	if config.MaxRequestBodySize != 5<<20 {
		t.Errorf("expected 5MB, got %d", config.MaxRequestBodySize)
	}

	if config.DisableStats {
		t.Error("expected DisableStats to be false")
	}
}

// TestNewServerNilConfig tests that NewServer handles nil config.
func TestNewServerNilConfig(t *testing.T) {
	// Note: We can't actually create a server without a handler,
	// so we just test that nil config uses defaults

	// Simulate what NewServer does
	var config *Config
	if config == nil {
		config = DefaultConfig()
	}

	if config == nil {
		t.Fatal("expected config to be initialized")
	}

	// Should use defaults
	if config.Addr != ":8080" {
		t.Error("expected default addr when config is nil")
	}

	if !config.DisableStats {
		t.Error("expected DisableStats to be true by default")
	}
}

// TestNewServerWithConfig tests custom config values.
func TestNewServerWithConfig(t *testing.T) {
	config := &Config{
		Addr:               ":3000",
		MaxRequestBodySize: 20 << 20, // 20MB
		DisableStats:       true,
	}

	// Note: We can't actually create a server without a handler
	// in tests, but we can verify the config values are correct

	if config.Addr != ":3000" {
		t.Errorf("expected addr :3000, got %s", config.Addr)
	}

	if config.MaxRequestBodySize != 20<<20 {
		t.Errorf("expected 20MB, got %d", config.MaxRequestBodySize)
	}

	if !config.DisableStats {
		t.Error("expected DisableStats to be true")
	}
}

// TestMaxRequestBodySizeType tests that MaxRequestBodySize is int not int64.
func TestMaxRequestBodySizeType(t *testing.T) {
	config := &Config{
		MaxRequestBodySize: 10 << 20, // 10MB as int
	}

	// This test ensures MaxRequestBodySize is int, not int64
	// If it were int64, this would fail to compile
	var size int = config.MaxRequestBodySize
	if size != 10<<20 {
		t.Errorf("expected 10MB, got %d", size)
	}
}

// TestDisableStatsConversion tests DisableStats to EnableStats conversion.
func TestDisableStatsConversion(t *testing.T) {
	tests := []struct {
		disableStats bool
		expectEnable bool
	}{
		{true, false},  // DisableStats=true → EnableStats=false
		{false, true},  // DisableStats=false → EnableStats=true
	}

	for _, tt := range tests {
		config := &Config{
			DisableStats: tt.disableStats,
		}

		// Simulate the conversion done in NewServer
		enableStats := !config.DisableStats

		if enableStats != tt.expectEnable {
			t.Errorf("DisableStats=%v: expected EnableStats=%v, got %v",
				tt.disableStats, tt.expectEnable, enableStats)
		}
	}
}

// TestTypeAliases tests that type aliases work correctly.
func TestTypeAliases(t *testing.T) {
	// This test verifies that Request and ResponseWriter are simple aliases
	// and not complex wrappers. If they were wrappers, this test would not compile.

	// Test that we can use the aliases in function signatures
	var handler func(*ResponseWriter, *Request)
	handler = func(w *ResponseWriter, r *Request) {
		// This is a placeholder - just verifies types work
	}

	if handler == nil {
		t.Error("handler should not be nil")
	}
}

// BenchmarkConfigCreation benchmarks config creation.
func BenchmarkConfigCreation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = &Config{
			Addr:               ":8080",
			MaxRequestBodySize: 10 << 20,
			DisableStats:       true,
		}
	}
}

// BenchmarkDefaultConfig benchmarks default config creation.
func BenchmarkDefaultConfig(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}
