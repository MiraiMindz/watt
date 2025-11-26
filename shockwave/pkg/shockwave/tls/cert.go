package tls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Certificate management for automatic TLS with Let's Encrypt

var (
	ErrCertNotFound     = errors.New("tls: certificate not found")
	ErrCertExpired      = errors.New("tls: certificate expired")
	ErrInvalidCert      = errors.New("tls: invalid certificate")
	ErrStorageFailed    = errors.New("tls: storage operation failed")
	ErrKeyGenerationErr = errors.New("tls: key generation failed")
)

// CertificateManager manages TLS certificates with automatic renewal
type CertificateManager struct {
	// Storage
	certDir     string
	cacheDir    string
	accountKey  crypto.PrivateKey
	accountPath string

	// Certificate cache
	mu          sync.RWMutex
	certificates map[string]*CertificateEntry

	// Configuration
	email          string
	renewBefore    time.Duration // Renew certificates this long before expiration
	checkInterval  time.Duration // How often to check for renewals
	staging        bool          // Use Let's Encrypt staging environment

	// ACME client
	acmeClient *ACMEClient

	// Renewal
	renewalChan chan string
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// CertificateEntry represents a cached certificate
type CertificateEntry struct {
	Certificate *tls.Certificate
	Leaf        *x509.Certificate
	Domains     []string
	IssuedAt    time.Time
	ExpiresAt   time.Time
	mu          sync.RWMutex
}

// CertManagerConfig holds configuration for the certificate manager
type CertManagerConfig struct {
	// Required
	Email   string   // Email for Let's Encrypt account
	Domains []string // Domains to manage certificates for

	// Optional
	CertDir       string        // Directory to store certificates (default: ./certs)
	Staging       bool          // Use Let's Encrypt staging environment
	RenewBefore   time.Duration // Renew before expiration (default: 30 days)
	CheckInterval time.Duration // Renewal check interval (default: 12 hours)
	KeyType       string        // Key type: "rsa2048", "rsa4096", "ecdsa256", "ecdsa384" (default: "ecdsa256")
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(config *CertManagerConfig) (*CertificateManager, error) {
	if config.Email == "" {
		return nil, errors.New("tls: email is required for Let's Encrypt")
	}

	if len(config.Domains) == 0 {
		return nil, errors.New("tls: at least one domain is required")
	}

	// Set defaults
	certDir := config.CertDir
	if certDir == "" {
		certDir = "./certs"
	}

	renewBefore := config.RenewBefore
	if renewBefore == 0 {
		renewBefore = 30 * 24 * time.Hour // 30 days
	}

	checkInterval := config.CheckInterval
	if checkInterval == 0 {
		checkInterval = 12 * time.Hour
	}

	// Create directories
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cert directory: %w", err)
	}

	accountPath := filepath.Join(certDir, "account.key")

	cm := &CertificateManager{
		certDir:       certDir,
		cacheDir:      certDir,
		accountPath:   accountPath,
		certificates:  make(map[string]*CertificateEntry),
		email:         config.Email,
		renewBefore:   renewBefore,
		checkInterval: checkInterval,
		staging:       config.Staging,
		renewalChan:   make(chan string, 10),
		stopChan:      make(chan struct{}),
	}

	// Load or create account key
	accountKey, err := cm.loadOrCreateAccountKey(config.KeyType)
	if err != nil {
		return nil, fmt.Errorf("failed to load account key: %w", err)
	}
	cm.accountKey = accountKey

	// Create ACME client
	cm.acmeClient, err = NewACMEClient(&ACMEConfig{
		Email:      config.Email,
		AccountKey: accountKey,
		Staging:    config.Staging,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ACME client: %w", err)
	}

	// Load existing certificates
	for _, domain := range config.Domains {
		if err := cm.loadCertificate(domain); err != nil {
			// Certificate doesn't exist yet, will be obtained on first use
			continue
		}
	}

	return cm, nil
}

// Start starts the certificate renewal monitor
func (cm *CertificateManager) Start() error {
	cm.wg.Add(1)
	go cm.renewalMonitor()
	return nil
}

// Stop stops the certificate manager
func (cm *CertificateManager) Stop() {
	close(cm.stopChan)
	cm.wg.Wait()
}

// GetCertificate returns a certificate for the given ClientHello
// This is the function used by crypto/tls.Config.GetCertificate
func (cm *CertificateManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	domain := hello.ServerName
	if domain == "" {
		return nil, errors.New("tls: no server name provided")
	}

	// Check cache first
	cm.mu.RLock()
	entry, exists := cm.certificates[domain]
	cm.mu.RUnlock()

	if exists {
		// Check if certificate is still valid
		if entry.IsValid() {
			return entry.Certificate, nil
		}
	}

	// Certificate doesn't exist or is invalid, obtain new one
	return cm.obtainCertificate(domain)
}

// obtainCertificate obtains a new certificate from Let's Encrypt
func (cm *CertificateManager) obtainCertificate(domain string) (*tls.Certificate, error) {
	// Generate private key for certificate
	privKey, err := cm.generateKey("ecdsa256")
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Create CSR
	csr, err := generateCSR(privKey, []string{domain})
	if err != nil {
		return nil, fmt.Errorf("failed to generate CSR: %w", err)
	}

	// Obtain certificate via ACME
	certChain, err := cm.acmeClient.ObtainCertificate(domain, csr)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain certificate: %w", err)
	}

	// Parse certificate
	cert, err := tls.X509KeyPair(certChain, encodePrivateKey(privKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Parse leaf certificate
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse leaf certificate: %w", err)
	}

	// Create entry
	entry := &CertificateEntry{
		Certificate: &cert,
		Leaf:        leaf,
		Domains:     []string{domain},
		IssuedAt:    leaf.NotBefore,
		ExpiresAt:   leaf.NotAfter,
	}

	// Store certificate
	if err := cm.storeCertificate(domain, &cert, privKey); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	// Cache certificate
	cm.mu.Lock()
	cm.certificates[domain] = entry
	cm.mu.Unlock()

	return &cert, nil
}

// loadCertificate loads a certificate from disk
func (cm *CertificateManager) loadCertificate(domain string) error {
	certPath := filepath.Join(cm.certDir, domain+".crt")
	keyPath := filepath.Join(cm.certDir, domain+".key")

	// Check if files exist
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return ErrCertNotFound
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return ErrCertNotFound
	}

	// Load certificate
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	// Parse leaf certificate
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse leaf certificate: %w", err)
	}

	// Create entry
	entry := &CertificateEntry{
		Certificate: &cert,
		Leaf:        leaf,
		Domains:     []string{domain},
		IssuedAt:    leaf.NotBefore,
		ExpiresAt:   leaf.NotAfter,
	}

	// Cache certificate
	cm.mu.Lock()
	cm.certificates[domain] = entry
	cm.mu.Unlock()

	return nil
}

// storeCertificate stores a certificate to disk
func (cm *CertificateManager) storeCertificate(domain string, cert *tls.Certificate, privKey crypto.PrivateKey) error {
	certPath := filepath.Join(cm.certDir, domain+".crt")
	keyPath := filepath.Join(cm.certDir, domain+".key")

	// Write certificate
	certPEM := encodeCertificate(cert)
	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write private key
	keyPEM := encodePrivateKey(privKey)
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	return nil
}

// loadOrCreateAccountKey loads or creates the ACME account key
func (cm *CertificateManager) loadOrCreateAccountKey(keyType string) (crypto.PrivateKey, error) {
	// Try to load existing key
	if _, err := os.Stat(cm.accountPath); err == nil {
		keyPEM, err := os.ReadFile(cm.accountPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read account key: %w", err)
		}

		key, err := parsePrivateKey(keyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to parse account key: %w", err)
		}

		return key, nil
	}

	// Generate new key
	if keyType == "" {
		keyType = "ecdsa256"
	}

	key, err := cm.generateKey(keyType)
	if err != nil {
		return nil, err
	}

	// Save key
	keyPEM := encodePrivateKey(key)
	if err := os.WriteFile(cm.accountPath, keyPEM, 0600); err != nil {
		return nil, fmt.Errorf("failed to save account key: %w", err)
	}

	return key, nil
}

// generateKey generates a private key
func (cm *CertificateManager) generateKey(keyType string) (crypto.PrivateKey, error) {
	switch keyType {
	case "rsa2048":
		return rsa.GenerateKey(rand.Reader, 2048)
	case "rsa4096":
		return rsa.GenerateKey(rand.Reader, 4096)
	case "ecdsa256":
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "ecdsa384":
		return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	default:
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}
}

// renewalMonitor monitors certificates and renews them when needed
func (cm *CertificateManager) renewalMonitor() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.checkRenewals()
		case domain := <-cm.renewalChan:
			cm.renewCertificate(domain)
		case <-cm.stopChan:
			return
		}
	}
}

// checkRenewals checks all certificates and renews those that are expiring soon
func (cm *CertificateManager) checkRenewals() {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	now := time.Now()

	for domain, entry := range cm.certificates {
		if entry.NeedsRenewal(now, cm.renewBefore) {
			// Queue for renewal (non-blocking)
			select {
			case cm.renewalChan <- domain:
			default:
				// Channel full, will try again on next check
			}
		}
	}
}

// renewCertificate renews a certificate
func (cm *CertificateManager) renewCertificate(domain string) {
	// Obtain new certificate
	cert, err := cm.obtainCertificate(domain)
	if err != nil {
		// Log error in production
		fmt.Printf("Failed to renew certificate for %s: %v\n", domain, err)
		return
	}

	fmt.Printf("Successfully renewed certificate for %s\n", domain)
	_ = cert // Certificate is already cached by obtainCertificate
}

// IsValid returns whether the certificate is still valid
func (ce *CertificateEntry) IsValid() bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	now := time.Now()
	return now.After(ce.IssuedAt) && now.Before(ce.ExpiresAt)
}

// NeedsRenewal returns whether the certificate needs renewal
func (ce *CertificateEntry) NeedsRenewal(now time.Time, renewBefore time.Duration) bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	renewAt := ce.ExpiresAt.Add(-renewBefore)
	return now.After(renewAt)
}

// DaysUntilExpiry returns the number of days until the certificate expires
func (ce *CertificateEntry) DaysUntilExpiry() int {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	duration := time.Until(ce.ExpiresAt)
	return int(duration.Hours() / 24)
}

// Helper functions

// generateCSR generates a Certificate Signing Request
func generateCSR(privKey crypto.PrivateKey, domains []string) ([]byte, error) {
	template := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: domains[0]},
		DNSNames: domains,
	}

	return x509.CreateCertificateRequest(rand.Reader, template, privKey)
}

// encodePrivateKey encodes a private key to PEM format
func encodePrivateKey(key crypto.PrivateKey) []byte {
	var pemType string
	var keyBytes []byte

	switch k := key.(type) {
	case *rsa.PrivateKey:
		pemType = "RSA PRIVATE KEY"
		keyBytes = x509.MarshalPKCS1PrivateKey(k)
	case *ecdsa.PrivateKey:
		pemType = "EC PRIVATE KEY"
		var err error
		keyBytes, err = x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil
		}
	default:
		return nil
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  pemType,
		Bytes: keyBytes,
	})
}

// parsePrivateKey parses a PEM-encoded private key
func parsePrivateKey(pemData []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		return x509.ParsePKCS8PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}
}

// encodeCertificate encodes a certificate to PEM format
func encodeCertificate(cert *tls.Certificate) []byte {
	var certPEM []byte
	for _, c := range cert.Certificate {
		certPEM = append(certPEM, pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: c,
		})...)
	}
	return certPEM
}
