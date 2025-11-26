package tls

import (
	"crypto/tls"
	"errors"
	"fmt"
	"time"
)

// TLS configuration builder for Shockwave HTTP server
// Provides simple API for automatic Let's Encrypt certificates

// Config represents TLS configuration options
type Config struct {
	// Automatic certificate management
	AutoCert      bool
	Email         string
	Domains       []string
	CertDir       string
	Staging       bool // Use Let's Encrypt staging (for testing)
	RenewBefore   time.Duration
	CheckInterval time.Duration

	// Manual certificate configuration
	CertFile string
	KeyFile  string

	// Advanced TLS options
	MinVersion            uint16
	MaxVersion            uint16
	CipherSuites          []uint16
	PreferServerCiphers   bool
	SessionTicketsDisabled bool
	Renegotiation         tls.RenegotiationSupport
	ClientAuth            tls.ClientAuthType
	ClientCAs             []string

	// ALPN protocols
	NextProtos []string

	// Certificate manager (internal)
	certManager *CertificateManager
}

// Default cipher suites (strong, modern ciphers only)
var defaultCipherSuites = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
}

// NewConfig creates a new TLS configuration with sensible defaults
func NewConfig() *Config {
	return &Config{
		MinVersion:           tls.VersionTLS12,
		MaxVersion:           tls.VersionTLS13,
		CipherSuites:         defaultCipherSuites,
		PreferServerCiphers:  true,
		SessionTicketsDisabled: false,
		Renegotiation:        tls.RenegotiateNever,
		NextProtos:           []string{"h2", "http/1.1"},
		RenewBefore:          30 * 24 * time.Hour, // 30 days
		CheckInterval:        12 * time.Hour,
	}
}

// WithAutoCert enables automatic certificate management via Let's Encrypt
func (c *Config) WithAutoCert(email string, domains ...string) *Config {
	c.AutoCert = true
	c.Email = email
	c.Domains = domains
	return c
}

// WithStaging enables Let's Encrypt staging environment (for testing)
func (c *Config) WithStaging() *Config {
	c.Staging = true
	return c
}

// WithCertDir sets the directory for storing certificates
func (c *Config) WithCertDir(dir string) *Config {
	c.CertDir = dir
	return c
}

// WithManualCert sets manual certificate files
func (c *Config) WithManualCert(certFile, keyFile string) *Config {
	c.AutoCert = false
	c.CertFile = certFile
	c.KeyFile = keyFile
	return c
}

// WithMinTLSVersion sets the minimum TLS version
func (c *Config) WithMinTLSVersion(version uint16) *Config {
	c.MinVersion = version
	return c
}

// WithMaxTLSVersion sets the maximum TLS version
func (c *Config) WithMaxTLSVersion(version uint16) *Config {
	c.MaxVersion = version
	return c
}

// WithCipherSuites sets custom cipher suites
func (c *Config) WithCipherSuites(suites []uint16) *Config {
	c.CipherSuites = suites
	return c
}

// WithALPN sets ALPN protocols
func (c *Config) WithALPN(protos ...string) *Config {
	c.NextProtos = protos
	return c
}

// WithClientAuth enables client certificate authentication
func (c *Config) WithClientAuth(authType tls.ClientAuthType) *Config {
	c.ClientAuth = authType
	return c
}

// WithRenewBefore sets how long before expiration to renew certificates
func (c *Config) WithRenewBefore(duration time.Duration) *Config {
	c.RenewBefore = duration
	return c
}

// WithCheckInterval sets how often to check for certificate renewals
func (c *Config) WithCheckInterval(duration time.Duration) *Config {
	c.CheckInterval = duration
	return c
}

// Build creates a *tls.Config from the configuration
func (c *Config) Build() (*tls.Config, error) {
	if c.AutoCert {
		return c.buildAutoCert()
	}
	return c.buildManualCert()
}

// buildAutoCert builds TLS config with automatic certificate management
func (c *Config) buildAutoCert() (*tls.Config, error) {
	if c.Email == "" {
		return nil, errors.New("email is required for automatic certificates")
	}

	if len(c.Domains) == 0 {
		return nil, errors.New("at least one domain is required for automatic certificates")
	}

	// Create certificate manager
	certManager, err := NewCertificateManager(&CertManagerConfig{
		Email:         c.Email,
		Domains:       c.Domains,
		CertDir:       c.CertDir,
		Staging:       c.Staging,
		RenewBefore:   c.RenewBefore,
		CheckInterval: c.CheckInterval,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate manager: %w", err)
	}

	// Start automatic renewal
	if err := certManager.Start(); err != nil {
		return nil, fmt.Errorf("failed to start certificate manager: %w", err)
	}

	c.certManager = certManager

	// Build TLS config
	tlsConfig := &tls.Config{
		GetCertificate:         certManager.GetCertificate,
		MinVersion:             c.MinVersion,
		MaxVersion:             c.MaxVersion,
		CipherSuites:           c.CipherSuites,
		PreferServerCipherSuites: c.PreferServerCiphers,
		SessionTicketsDisabled: c.SessionTicketsDisabled,
		Renegotiation:          c.Renegotiation,
		NextProtos:             c.NextProtos,
		ClientAuth:             c.ClientAuth,
	}

	return tlsConfig, nil
}

// buildManualCert builds TLS config with manual certificate files
func (c *Config) buildManualCert() (*tls.Config, error) {
	if c.CertFile == "" || c.KeyFile == "" {
		return nil, errors.New("certificate and key files are required")
	}

	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates:           []tls.Certificate{cert},
		MinVersion:             c.MinVersion,
		MaxVersion:             c.MaxVersion,
		CipherSuites:           c.CipherSuites,
		PreferServerCipherSuites: c.PreferServerCiphers,
		SessionTicketsDisabled: c.SessionTicketsDisabled,
		Renegotiation:          c.Renegotiation,
		NextProtos:             c.NextProtos,
		ClientAuth:             c.ClientAuth,
	}

	return tlsConfig, nil
}

// Stop stops the certificate manager (if using auto-cert)
func (c *Config) Stop() {
	if c.certManager != nil {
		c.certManager.Stop()
	}
}

// QuickTLS creates a TLS config with automatic Let's Encrypt certificates
// This is the simplest way to get automatic HTTPS
func QuickTLS(email string, domains ...string) (*tls.Config, error) {
	config := NewConfig().WithAutoCert(email, domains...)
	return config.Build()
}

// QuickTLSStaging creates a TLS config with Let's Encrypt staging (for testing)
func QuickTLSStaging(email string, domains ...string) (*tls.Config, error) {
	config := NewConfig().
		WithAutoCert(email, domains...).
		WithStaging()
	return config.Build()
}

// ManualTLS creates a TLS config with manual certificate files
func ManualTLS(certFile, keyFile string) (*tls.Config, error) {
	config := NewConfig().WithManualCert(certFile, keyFile)
	return config.Build()
}

// SecureDefaults returns a TLS config with secure default settings
// Requires TLS 1.2+, strong ciphers only, perfect forward secrecy
func SecureDefaults() *Config {
	return &Config{
		MinVersion:  tls.VersionTLS12,
		MaxVersion:  tls.VersionTLS13,
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites (implicit, can't be configured)
			// TLS 1.2 cipher suites with PFS
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		PreferServerCiphers:    true,
		SessionTicketsDisabled: false,
		Renegotiation:          tls.RenegotiateNever,
		NextProtos:             []string{"h2", "http/1.1"},
	}
}

// HTTP3Defaults returns a TLS config optimized for HTTP/3
func HTTP3Defaults() *Config {
	config := SecureDefaults()
	config.NextProtos = []string{"h3", "h2", "http/1.1"}
	config.MinVersion = tls.VersionTLS13 // HTTP/3 requires TLS 1.3
	return config
}

// GetCertificateInfo returns information about managed certificates
func (c *Config) GetCertificateInfo() map[string]*CertificateEntry {
	if c.certManager == nil {
		return nil
	}

	c.certManager.mu.RLock()
	defer c.certManager.mu.RUnlock()

	// Return a copy to avoid race conditions
	info := make(map[string]*CertificateEntry, len(c.certManager.certificates))
	for domain, entry := range c.certManager.certificates {
		info[domain] = entry
	}

	return info
}

// RenewCertificate manually triggers certificate renewal for a domain
func (c *Config) RenewCertificate(domain string) error {
	if c.certManager == nil {
		return errors.New("certificate manager not initialized")
	}

	// Queue renewal
	select {
	case c.certManager.renewalChan <- domain:
		return nil
	default:
		return errors.New("renewal queue full")
	}
}

// Example usage documentation
const ExampleUsage = `
// Simplest usage - automatic HTTPS with Let's Encrypt
tlsConfig, err := tls.QuickTLS("admin@example.com", "example.com", "www.example.com")
if err != nil {
	log.Fatal(err)
}

// Or with more control
config := tls.NewConfig().
	WithAutoCert("admin@example.com", "example.com").
	WithCertDir("/etc/letsencrypt/live").
	WithRenewBefore(30 * 24 * time.Hour).
	WithMinTLSVersion(tls.VersionTLS12)

tlsConfig, err := config.Build()
if err != nil {
	log.Fatal(err)
}

// Use with HTTP server
server := &http.Server{
	Addr:      ":443",
	TLSConfig: tlsConfig,
	Handler:   myHandler,
}

log.Fatal(server.ListenAndServeTLS("", ""))

// For testing with staging environment
tlsConfig, err := tls.QuickTLSStaging("admin@example.com", "example.com")

// Manual certificates
tlsConfig, err := tls.ManualTLS("/path/to/cert.pem", "/path/to/key.pem")

// HTTP/3 optimized
config := tls.HTTP3Defaults().
	WithAutoCert("admin@example.com", "example.com")

// Check certificate status
info := config.GetCertificateInfo()
for domain, cert := range info {
	fmt.Printf("%s expires in %d days\n", domain, cert.DaysUntilExpiry())
}

// Manually renew a certificate
err := config.RenewCertificate("example.com")
`
