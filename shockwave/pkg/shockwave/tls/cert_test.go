package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test certificate generation
func TestGenerateKey(t *testing.T) {
	cm := &CertificateManager{}

	keyTypes := []string{"rsa2048", "rsa4096", "ecdsa256", "ecdsa384"}

	for _, keyType := range keyTypes {
		key, err := cm.generateKey(keyType)
		if err != nil {
			t.Fatalf("Failed to generate %s key: %v", keyType, err)
		}

		if key == nil {
			t.Errorf("Generated key is nil for type %s", keyType)
		}
	}

	t.Logf("✓ Key generation working for all types")
}

// Test private key encoding/decoding
func TestPrivateKeyEncoding(t *testing.T) {
	// Generate test key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Encode key
	encoded := encodePrivateKey(key)
	if encoded == nil {
		t.Fatal("Encoded key is nil")
	}

	// Decode key
	decoded, err := parsePrivateKey(encoded)
	if err != nil {
		t.Fatalf("Failed to parse key: %v", err)
	}

	// Verify key matches
	ecKey, ok := decoded.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatal("Decoded key is not ECDSA")
	}

	if ecKey.D.Cmp(key.D) != 0 {
		t.Error("Decoded key doesn't match original")
	}

	t.Logf("✓ Private key encoding/decoding working correctly")
}

// Test CSR generation
func TestGenerateCSR(t *testing.T) {
	// Generate test key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Generate CSR
	domains := []string{"example.com", "www.example.com"}
	csr, err := generateCSR(key, domains)
	if err != nil {
		t.Fatalf("Failed to generate CSR: %v", err)
	}

	// Parse CSR
	parsedCSR, err := x509.ParseCertificateRequest(csr)
	if err != nil {
		t.Fatalf("Failed to parse CSR: %v", err)
	}

	// Verify domains
	if parsedCSR.Subject.CommonName != domains[0] {
		t.Errorf("Expected CN %s, got %s", domains[0], parsedCSR.Subject.CommonName)
	}

	if len(parsedCSR.DNSNames) != len(domains) {
		t.Errorf("Expected %d DNS names, got %d", len(domains), len(parsedCSR.DNSNames))
	}

	t.Logf("✓ CSR generation working correctly")
}

// Test certificate storage and loading
func TestCertificateStorage(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "cert-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificate
	key, cert := generateTestCertificate(t, "test.example.com")

	// Create certificate manager
	cm := &CertificateManager{
		certDir:      tmpDir,
		certificates: make(map[string]*CertificateEntry),
	}

	// Store certificate
	domain := "test.example.com"
	if err := cm.storeCertificate(domain, cert, key); err != nil {
		t.Fatalf("Failed to store certificate: %v", err)
	}

	// Verify files exist
	certPath := filepath.Join(tmpDir, domain+".crt")
	keyPath := filepath.Join(tmpDir, domain+".key")

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("Certificate file not created")
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("Key file not created")
	}

	// Load certificate
	if err := cm.loadCertificate(domain); err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}

	// Verify loaded certificate
	entry, exists := cm.certificates[domain]
	if !exists {
		t.Fatal("Certificate not in cache")
	}

	if entry.Certificate == nil {
		t.Error("Loaded certificate is nil")
	}

	if len(entry.Domains) != 1 || entry.Domains[0] != domain {
		t.Errorf("Expected domain %s, got %v", domain, entry.Domains)
	}

	t.Logf("✓ Certificate storage and loading working correctly")
}

// Test certificate entry validation
func TestCertificateEntryValidation(t *testing.T) {
	now := time.Now()

	// Create valid certificate entry
	validEntry := &CertificateEntry{
		IssuedAt:  now.Add(-24 * time.Hour),
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}

	if !validEntry.IsValid() {
		t.Error("Valid certificate reported as invalid")
	}

	// Create expired certificate entry
	expiredEntry := &CertificateEntry{
		IssuedAt:  now.Add(-90 * 24 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
	}

	if expiredEntry.IsValid() {
		t.Error("Expired certificate reported as valid")
	}

	// Create future certificate entry
	futureEntry := &CertificateEntry{
		IssuedAt:  now.Add(24 * time.Hour),
		ExpiresAt: now.Add(90 * 24 * time.Hour),
	}

	if futureEntry.IsValid() {
		t.Error("Future certificate reported as valid")
	}

	t.Logf("✓ Certificate validation working correctly")
}

// Test certificate renewal checking
func TestCertificateRenewal(t *testing.T) {
	now := time.Now()

	// Certificate expiring in 60 days
	entry := &CertificateEntry{
		IssuedAt:  now.Add(-30 * 24 * time.Hour),
		ExpiresAt: now.Add(60 * 24 * time.Hour),
	}

	// Should need renewal if renewing 90 days before expiry
	renewBefore := 90 * 24 * time.Hour
	if !entry.NeedsRenewal(now, renewBefore) {
		t.Error("Certificate should need renewal (60 days < 90 days)")
	}

	// Should NOT need renewal if renewing 30 days before expiry
	renewBefore = 30 * 24 * time.Hour
	if entry.NeedsRenewal(now, renewBefore) {
		t.Error("Certificate should not need renewal (60 days > 30 days)")
	}

	// Test DaysUntilExpiry
	days := entry.DaysUntilExpiry()
	if days < 59 || days > 61 {
		t.Errorf("Expected ~60 days until expiry, got %d", days)
	}

	t.Logf("✓ Certificate renewal checking working correctly")
}

// Test account key management
func TestAccountKeyManagement(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "account-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	accountPath := filepath.Join(tmpDir, "account.key")

	cm := &CertificateManager{
		accountPath: accountPath,
	}

	// Load or create account key (should create)
	key1, err := cm.loadOrCreateAccountKey("ecdsa256")
	if err != nil {
		t.Fatalf("Failed to create account key: %v", err)
	}

	if key1 == nil {
		t.Fatal("Account key is nil")
	}

	// Verify key file exists
	if _, err := os.Stat(accountPath); os.IsNotExist(err) {
		t.Error("Account key file not created")
	}

	// Load account key again (should load existing)
	key2, err := cm.loadOrCreateAccountKey("ecdsa256")
	if err != nil {
		t.Fatalf("Failed to load account key: %v", err)
	}

	// Verify keys match
	ecKey1, ok1 := key1.(*ecdsa.PrivateKey)
	ecKey2, ok2 := key2.(*ecdsa.PrivateKey)

	if !ok1 || !ok2 {
		t.Fatal("Keys are not ECDSA")
	}

	if ecKey1.D.Cmp(ecKey2.D) != 0 {
		t.Error("Loaded key doesn't match created key")
	}

	t.Logf("✓ Account key management working correctly")
}

// Test certificate encoding
func TestCertificateEncoding(t *testing.T) {
	_, cert := generateTestCertificate(t, "test.example.com")

	// Encode certificate
	encoded := encodeCertificate(cert)
	if encoded == nil {
		t.Fatal("Encoded certificate is nil")
	}

	// Verify PEM format
	block, _ := pem.Decode(encoded)
	if block == nil {
		t.Fatal("Failed to decode PEM")
	}

	if block.Type != "CERTIFICATE" {
		t.Errorf("Expected CERTIFICATE block, got %s", block.Type)
	}

	// Parse certificate
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	if parsed.Subject.CommonName != "test.example.com" {
		t.Errorf("Expected CN test.example.com, got %s", parsed.Subject.CommonName)
	}

	t.Logf("✓ Certificate encoding working correctly")
}

// Helper functions

// generateTestCertificate generates a self-signed certificate for testing
func generateTestCertificate(t *testing.T, domain string) (*ecdsa.PrivateKey, *tls.Certificate) {
	// Generate key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: domain,
		},
		DNSNames:              []string{domain},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(90 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Create tls.Certificate
	cert := &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	return key, cert
}

// Benchmark tests

func BenchmarkKeyGeneration(b *testing.B) {
	cm := &CertificateManager{}

	b.Run("ECDSA-P256", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			cm.generateKey("ecdsa256")
		}
	})

	b.Run("RSA-2048", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			cm.generateKey("rsa2048")
		}
	})
}

func BenchmarkCSRGeneration(b *testing.B) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	domains := []string{"example.com"}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		generateCSR(key, domains)
	}
}

func BenchmarkPrivateKeyEncoding(b *testing.B) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		encodePrivateKey(key)
	}
}

func BenchmarkPrivateKeyParsing(b *testing.B) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	encoded := encodePrivateKey(key)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parsePrivateKey(encoded)
	}
}
