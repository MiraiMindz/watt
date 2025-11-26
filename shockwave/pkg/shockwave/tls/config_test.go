package tls

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test TLS config creation
func TestNewConfig(t *testing.T) {
	config := NewConfig()

	if config == nil {
		t.Fatal("NewConfig returned nil")
	}

	// Verify defaults
	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion TLS 1.2, got 0x%x", config.MinVersion)
	}

	if config.MaxVersion != tls.VersionTLS13 {
		t.Errorf("Expected MaxVersion TLS 1.3, got 0x%x", config.MaxVersion)
	}

	if !config.PreferServerCiphers {
		t.Error("Expected PreferServerCiphers to be true")
	}

	if config.Renegotiation != tls.RenegotiateNever {
		t.Error("Expected Renegotiation to be Never")
	}

	if len(config.NextProtos) == 0 {
		t.Error("Expected ALPN protocols to be set")
	}

	t.Logf("✓ Config creation with defaults working correctly")
}

// Test config builder pattern
func TestConfigBuilder(t *testing.T) {
	config := NewConfig().
		WithMinTLSVersion(tls.VersionTLS13).
		WithMaxTLSVersion(tls.VersionTLS13).
		WithALPN("h3", "h2").
		WithRenewBefore(15 * 24 * time.Hour).
		WithCheckInterval(6 * time.Hour)

	if config.MinVersion != tls.VersionTLS13 {
		t.Error("MinVersion not set correctly")
	}

	if config.MaxVersion != tls.VersionTLS13 {
		t.Error("MaxVersion not set correctly")
	}

	if len(config.NextProtos) != 2 || config.NextProtos[0] != "h3" {
		t.Error("ALPN not set correctly")
	}

	if config.RenewBefore != 15*24*time.Hour {
		t.Error("RenewBefore not set correctly")
	}

	if config.CheckInterval != 6*time.Hour {
		t.Error("CheckInterval not set correctly")
	}

	t.Logf("✓ Config builder pattern working correctly")
}

// Test secure defaults
func TestSecureDefaults(t *testing.T) {
	config := SecureDefaults()

	// Verify TLS version requirements
	if config.MinVersion < tls.VersionTLS12 {
		t.Error("Secure defaults should require TLS 1.2+")
	}

	// Verify cipher suites are strong
	if len(config.CipherSuites) == 0 {
		t.Error("Secure defaults should have cipher suites")
	}

	// All cipher suites should support PFS (ECDHE)
	for _, suite := range config.CipherSuites {
		switch suite {
		case tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305:
			// Good - supports PFS
		default:
			t.Errorf("Cipher suite 0x%x does not support PFS", suite)
		}
	}

	// Verify server cipher preference
	if !config.PreferServerCiphers {
		t.Error("Secure defaults should prefer server ciphers")
	}

	// Verify no renegotiation
	if config.Renegotiation != tls.RenegotiateNever {
		t.Error("Secure defaults should disable renegotiation")
	}

	t.Logf("✓ Secure defaults configuration working correctly")
}

// Test HTTP/3 defaults
func TestHTTP3Defaults(t *testing.T) {
	config := HTTP3Defaults()

	// Verify TLS 1.3 requirement
	if config.MinVersion != tls.VersionTLS13 {
		t.Error("HTTP/3 requires TLS 1.3")
	}

	// Verify h3 ALPN
	found := false
	for _, proto := range config.NextProtos {
		if proto == "h3" {
			found = true
			break
		}
	}

	if !found {
		t.Error("HTTP/3 defaults should include h3 ALPN")
	}

	t.Logf("✓ HTTP/3 defaults configuration working correctly")
}

// Test manual certificate loading
func TestManualCertBuild(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificate
	key, cert := generateTestCertificate(t, "test.example.com")

	// Save certificate and key
	certPath := filepath.Join(tmpDir, "test.crt")
	keyPath := filepath.Join(tmpDir, "test.key")

	certPEM := encodeCertificate(cert)
	keyPEM := encodePrivateKey(key)

	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		t.Fatalf("Failed to write cert: %v", err)
	}

	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		t.Fatalf("Failed to write key: %v", err)
	}

	// Build config
	config := NewConfig().WithManualCert(certPath, keyPath)
	tlsConfig, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	if tlsConfig == nil {
		t.Fatal("TLS config is nil")
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Errorf("Expected 1 certificate, got %d", len(tlsConfig.Certificates))
	}

	t.Logf("✓ Manual certificate loading working correctly")
}

// Test manual certificate with missing files
func TestManualCertMissingFiles(t *testing.T) {
	config := NewConfig().WithManualCert("/nonexistent/cert.pem", "/nonexistent/key.pem")
	_, err := config.Build()

	if err == nil {
		t.Error("Expected error for missing certificate files")
	}

	t.Logf("✓ Missing certificate files properly handled")
}

// Test ManualTLS helper
func TestManualTLSHelper(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "manual-tls-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificate
	key, cert := generateTestCertificate(t, "test.example.com")

	// Save certificate and key
	certPath := filepath.Join(tmpDir, "test.crt")
	keyPath := filepath.Join(tmpDir, "test.key")

	certPEM := encodeCertificate(cert)
	keyPEM := encodePrivateKey(key)

	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		t.Fatalf("Failed to write cert: %v", err)
	}

	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		t.Fatalf("Failed to write key: %v", err)
	}

	// Use ManualTLS helper
	tlsConfig, err := ManualTLS(certPath, keyPath)
	if err != nil {
		t.Fatalf("ManualTLS failed: %v", err)
	}

	if tlsConfig == nil {
		t.Fatal("TLS config is nil")
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Error("Expected 1 certificate")
	}

	t.Logf("✓ ManualTLS helper working correctly")
}

// Test cipher suite validation
func TestCipherSuiteConfiguration(t *testing.T) {
	customSuites := []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	}

	config := NewConfig().WithCipherSuites(customSuites)
	tlsConfig, err := ManualTLS("/dev/null", "/dev/null") // Will fail, but tests cipher config
	_ = tlsConfig
	_ = err

	if len(config.CipherSuites) != len(customSuites) {
		t.Error("Cipher suites not set correctly")
	}

	for i, suite := range config.CipherSuites {
		if suite != customSuites[i] {
			t.Errorf("Cipher suite %d mismatch", i)
		}
	}

	t.Logf("✓ Custom cipher suite configuration working correctly")
}

// Test default cipher suites
func TestDefaultCipherSuites(t *testing.T) {
	if len(defaultCipherSuites) == 0 {
		t.Error("Default cipher suites should not be empty")
	}

	// Verify all are strong modern ciphers
	for _, suite := range defaultCipherSuites {
		switch suite {
		case tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305:
			// Good cipher
		default:
			t.Errorf("Weak or unknown cipher in defaults: 0x%x", suite)
		}
	}

	t.Logf("✓ Default cipher suites are strong and modern")
}

// Test client auth configuration
func TestClientAuthConfiguration(t *testing.T) {
	authTypes := []tls.ClientAuthType{
		tls.NoClientCert,
		tls.RequestClientCert,
		tls.RequireAnyClientCert,
		tls.VerifyClientCertIfGiven,
		tls.RequireAndVerifyClientCert,
	}

	for _, authType := range authTypes {
		config := NewConfig().WithClientAuth(authType)

		if config.ClientAuth != authType {
			t.Errorf("ClientAuth not set correctly: expected %v, got %v", authType, config.ClientAuth)
		}
	}

	t.Logf("✓ Client authentication configuration working correctly")
}

// Test TLS version configuration
func TestTLSVersionConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		minVersion uint16
		maxVersion uint16
		valid      bool
	}{
		{"TLS 1.2 to 1.3", tls.VersionTLS12, tls.VersionTLS13, true},
		{"TLS 1.3 only", tls.VersionTLS13, tls.VersionTLS13, true},
		{"TLS 1.2 only", tls.VersionTLS12, tls.VersionTLS12, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig().
				WithMinTLSVersion(tt.minVersion).
				WithMaxTLSVersion(tt.maxVersion)

			if config.MinVersion != tt.minVersion {
				t.Errorf("MinVersion not set: expected 0x%x, got 0x%x", tt.minVersion, config.MinVersion)
			}

			if config.MaxVersion != tt.maxVersion {
				t.Errorf("MaxVersion not set: expected 0x%x, got 0x%x", tt.maxVersion, config.MaxVersion)
			}
		})
	}

	t.Logf("✓ TLS version configuration working correctly")
}

// Test ALPN configuration
func TestALPNConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		protos []string
	}{
		{"HTTP/2 and HTTP/1.1", []string{"h2", "http/1.1"}},
		{"HTTP/3 priority", []string{"h3", "h2", "http/1.1"}},
		{"HTTP/1.1 only", []string{"http/1.1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig().WithALPN(tt.protos...)

			if len(config.NextProtos) != len(tt.protos) {
				t.Errorf("Expected %d protocols, got %d", len(tt.protos), len(config.NextProtos))
			}

			for i, proto := range tt.protos {
				if config.NextProtos[i] != proto {
					t.Errorf("Protocol %d mismatch: expected %s, got %s", i, proto, config.NextProtos[i])
				}
			}
		})
	}

	t.Logf("✓ ALPN configuration working correctly")
}

// Test config stop (should not panic)
func TestConfigStop(t *testing.T) {
	config := NewConfig()

	// Should not panic even if certManager is nil
	config.Stop()

	t.Logf("✓ Config stop working correctly")
}

// Test certificate info retrieval
func TestGetCertificateInfo(t *testing.T) {
	config := NewConfig()

	// Should return nil if certManager is not initialized
	info := config.GetCertificateInfo()
	if info != nil {
		t.Error("Expected nil for uninitialized certManager")
	}

	t.Logf("✓ Certificate info retrieval working correctly")
}

// Test manual renewal
func TestManualRenewal(t *testing.T) {
	config := NewConfig()

	// Should return error if certManager is not initialized
	err := config.RenewCertificate("example.com")
	if err == nil {
		t.Error("Expected error for uninitialized certManager")
	}

	t.Logf("✓ Manual renewal error handling working correctly")
}

// Benchmark tests

func BenchmarkConfigCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		NewConfig()
	}
}

func BenchmarkSecureDefaults(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		SecureDefaults()
	}
}

func BenchmarkHTTP3Defaults(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		HTTP3Defaults()
	}
}
