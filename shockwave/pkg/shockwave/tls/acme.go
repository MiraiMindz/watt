package tls

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ACME (Automatic Certificate Management Environment) client for Let's Encrypt
// Implements RFC 8555

const (
	// Let's Encrypt production directory
	LEProductionURL = "https://acme-v02.api.letsencrypt.org/directory"
	// Let's Encrypt staging directory (for testing)
	LEStagingURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

var (
	ErrACMEFailed          = errors.New("acme: request failed")
	ErrChallengeFailed     = errors.New("acme: challenge failed")
	ErrChallengeTimeout    = errors.New("acme: challenge timeout")
	ErrOrderNotReady       = errors.New("acme: order not ready")
	ErrAuthorizationFailed = errors.New("acme: authorization failed")
)

// ACMEClient handles communication with ACME servers (Let's Encrypt)
type ACMEClient struct {
	// Configuration
	directoryURL string
	email        string
	accountKey   crypto.PrivateKey
	staging      bool

	// ACME directory endpoints
	directory *ACMEDirectory
	mu        sync.RWMutex

	// Account
	accountURL string
	kid        string // Key ID

	// HTTP client
	httpClient *http.Client

	// Challenge server for HTTP-01
	challengeServer   *http.Server
	challengeTokens   map[string]string
	challengeTokensMu sync.RWMutex
}

// ACMEConfig holds configuration for the ACME client
type ACMEConfig struct {
	Email      string
	AccountKey crypto.PrivateKey
	Staging    bool
}

// ACMEDirectory represents the ACME directory structure
type ACMEDirectory struct {
	NewNonce   string `json:"newNonce"`
	NewAccount string `json:"newAccount"`
	NewOrder   string `json:"newOrder"`
	RevokeCert string `json:"revokeCert"`
	KeyChange  string `json:"keyChange"`
}

// ACMEAccount represents an ACME account
type ACMEAccount struct {
	Status  string   `json:"status"`
	Contact []string `json:"contact"`
	Orders  string   `json:"orders"`
}

// ACMEOrder represents an ACME order
type ACMEOrder struct {
	Status         string          `json:"status"`
	Expires        string          `json:"expires"`
	Identifiers    []ACMEIdentifier `json:"identifiers"`
	Authorizations []string        `json:"authorizations"`
	Finalize       string          `json:"finalize"`
	Certificate    string          `json:"certificate,omitempty"`
}

// ACMEIdentifier represents a domain identifier
type ACMEIdentifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ACMEAuthorization represents an authorization challenge
type ACMEAuthorization struct {
	Status     string           `json:"status"`
	Expires    string           `json:"expires"`
	Identifier ACMEIdentifier   `json:"identifier"`
	Challenges []ACMEChallenge  `json:"challenges"`
}

// ACMEChallenge represents a challenge
type ACMEChallenge struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	URL    string `json:"url"`
	Token  string `json:"token"`
}

// NewACMEClient creates a new ACME client
func NewACMEClient(config *ACMEConfig) (*ACMEClient, error) {
	if config.Email == "" {
		return nil, errors.New("acme: email is required")
	}

	if config.AccountKey == nil {
		return nil, errors.New("acme: account key is required")
	}

	directoryURL := LEProductionURL
	if config.Staging {
		directoryURL = LEStagingURL
	}

	client := &ACMEClient{
		directoryURL:    directoryURL,
		email:           config.Email,
		accountKey:      config.AccountKey,
		staging:         config.Staging,
		challengeTokens: make(map[string]string),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Fetch directory
	if err := client.fetchDirectory(); err != nil {
		return nil, fmt.Errorf("failed to fetch ACME directory: %w", err)
	}

	// Register or get account
	if err := client.registerAccount(); err != nil {
		return nil, fmt.Errorf("failed to register account: %w", err)
	}

	// Start challenge server
	if err := client.startChallengeServer(); err != nil {
		return nil, fmt.Errorf("failed to start challenge server: %w", err)
	}

	return client, nil
}

// ObtainCertificate obtains a certificate for the given domain
func (ac *ACMEClient) ObtainCertificate(domain string, csr []byte) ([]byte, error) {
	// Create order
	order, err := ac.createOrder([]string{domain})
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Process authorizations
	for _, authURL := range order.Authorizations {
		if err := ac.processAuthorization(authURL); err != nil {
			return nil, fmt.Errorf("authorization failed: %w", err)
		}
	}

	// Finalize order
	if err := ac.finalizeOrder(order, csr); err != nil {
		return nil, fmt.Errorf("failed to finalize order: %w", err)
	}

	// Download certificate
	cert, err := ac.downloadCertificate(order)
	if err != nil {
		return nil, fmt.Errorf("failed to download certificate: %w", err)
	}

	return cert, nil
}

// fetchDirectory fetches the ACME directory
func (ac *ACMEClient) fetchDirectory() error {
	resp, err := ac.httpClient.Get(ac.directoryURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var directory ACMEDirectory
	if err := json.NewDecoder(resp.Body).Decode(&directory); err != nil {
		return err
	}

	ac.mu.Lock()
	ac.directory = &directory
	ac.mu.Unlock()

	return nil
}

// registerAccount registers or retrieves an existing account
func (ac *ACMEClient) registerAccount() error {
	nonce, err := ac.getNonce()
	if err != nil {
		return err
	}

	// Create account registration payload
	payload := map[string]interface{}{
		"termsOfServiceAgreed": true,
		"contact":              []string{"mailto:" + ac.email},
	}

	// Sign request
	signed, err := ac.signRequest(ac.directory.NewAccount, "", nonce, payload)
	if err != nil {
		return err
	}

	// Send request
	resp, err := ac.httpClient.Post(ac.directory.NewAccount, "application/jose+json", bytes.NewReader(signed))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Accept both 201 (created) and 200 (existing account)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("account registration failed: %d - %s", resp.StatusCode, string(body))
	}

	// Store account URL
	ac.accountURL = resp.Header.Get("Location")
	ac.kid = ac.accountURL

	return nil
}

// createOrder creates a new certificate order
func (ac *ACMEClient) createOrder(domains []string) (*ACMEOrder, error) {
	nonce, err := ac.getNonce()
	if err != nil {
		return nil, err
	}

	// Create identifiers
	identifiers := make([]ACMEIdentifier, len(domains))
	for i, domain := range domains {
		identifiers[i] = ACMEIdentifier{
			Type:  "dns",
			Value: domain,
		}
	}

	payload := map[string]interface{}{
		"identifiers": identifiers,
	}

	// Sign request
	signed, err := ac.signRequest(ac.directory.NewOrder, ac.kid, nonce, payload)
	if err != nil {
		return nil, err
	}

	// Send request
	resp, err := ac.httpClient.Post(ac.directory.NewOrder, "application/jose+json", bytes.NewReader(signed))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("order creation failed: %d - %s", resp.StatusCode, string(body))
	}

	var order ACMEOrder
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		return nil, err
	}

	return &order, nil
}

// processAuthorization processes an authorization challenge
func (ac *ACMEClient) processAuthorization(authURL string) error {
	// Get authorization
	auth, err := ac.getAuthorization(authURL)
	if err != nil {
		return err
	}

	// Find HTTP-01 challenge
	var httpChallenge *ACMEChallenge
	for i := range auth.Challenges {
		if auth.Challenges[i].Type == "http-01" {
			httpChallenge = &auth.Challenges[i]
			break
		}
	}

	if httpChallenge == nil {
		return errors.New("no http-01 challenge found")
	}

	// Setup challenge response
	keyAuth, err := ac.getKeyAuthorization(httpChallenge.Token)
	if err != nil {
		return err
	}

	ac.challengeTokensMu.Lock()
	ac.challengeTokens[httpChallenge.Token] = keyAuth
	ac.challengeTokensMu.Unlock()

	defer func() {
		ac.challengeTokensMu.Lock()
		delete(ac.challengeTokens, httpChallenge.Token)
		ac.challengeTokensMu.Unlock()
	}()

	// Notify ACME server that challenge is ready
	if err := ac.notifyChallenge(httpChallenge.URL); err != nil {
		return err
	}

	// Wait for challenge validation
	if err := ac.waitForChallenge(authURL); err != nil {
		return err
	}

	return nil
}

// getAuthorization retrieves an authorization
func (ac *ACMEClient) getAuthorization(url string) (*ACMEAuthorization, error) {
	nonce, err := ac.getNonce()
	if err != nil {
		return nil, err
	}

	// Sign request (POST-as-GET)
	signed, err := ac.signRequest(url, ac.kid, nonce, "")
	if err != nil {
		return nil, err
	}

	resp, err := ac.httpClient.Post(url, "application/jose+json", bytes.NewReader(signed))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get authorization failed: %d - %s", resp.StatusCode, string(body))
	}

	var auth ACMEAuthorization
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return nil, err
	}

	return &auth, nil
}

// notifyChallenge notifies the ACME server that the challenge is ready
func (ac *ACMEClient) notifyChallenge(challengeURL string) error {
	nonce, err := ac.getNonce()
	if err != nil {
		return err
	}

	// Empty payload to trigger challenge
	payload := map[string]interface{}{}

	signed, err := ac.signRequest(challengeURL, ac.kid, nonce, payload)
	if err != nil {
		return err
	}

	resp, err := ac.httpClient.Post(challengeURL, "application/jose+json", bytes.NewReader(signed))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("challenge notification failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// waitForChallenge waits for the challenge to be validated
func (ac *ACMEClient) waitForChallenge(authURL string) error {
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return ErrChallengeTimeout
		case <-ticker.C:
			auth, err := ac.getAuthorization(authURL)
			if err != nil {
				return err
			}

			if auth.Status == "valid" {
				return nil
			}

			if auth.Status == "invalid" {
				return ErrChallengeFailed
			}

			// Status is still "pending", continue waiting
		}
	}
}

// finalizeOrder finalizes the order with the CSR
func (ac *ACMEClient) finalizeOrder(order *ACMEOrder, csr []byte) error {
	nonce, err := ac.getNonce()
	if err != nil {
		return err
	}

	// Encode CSR
	csrEncoded := base64.RawURLEncoding.EncodeToString(csr)

	payload := map[string]interface{}{
		"csr": csrEncoded,
	}

	signed, err := ac.signRequest(order.Finalize, ac.kid, nonce, payload)
	if err != nil {
		return err
	}

	resp, err := ac.httpClient.Post(order.Finalize, "application/jose+json", bytes.NewReader(signed))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("finalize failed: %d - %s", resp.StatusCode, string(body))
	}

	// Update order
	if err := json.NewDecoder(resp.Body).Decode(order); err != nil {
		return err
	}

	// Wait for order to be ready
	return ac.waitForOrder(order)
}

// waitForOrder waits for the order to be ready
func (ac *ACMEClient) waitForOrder(order *ACMEOrder) error {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return ErrChallengeTimeout
		case <-ticker.C:
			if order.Status == "valid" && order.Certificate != "" {
				return nil
			}

			if order.Status == "invalid" {
				return ErrOrderNotReady
			}

			// Status is still "processing", continue waiting
			time.Sleep(2 * time.Second)
		}
	}
}

// downloadCertificate downloads the issued certificate
func (ac *ACMEClient) downloadCertificate(order *ACMEOrder) ([]byte, error) {
	nonce, err := ac.getNonce()
	if err != nil {
		return nil, err
	}

	// Sign request (POST-as-GET)
	signed, err := ac.signRequest(order.Certificate, ac.kid, nonce, "")
	if err != nil {
		return nil, err
	}

	resp, err := ac.httpClient.Post(order.Certificate, "application/jose+json", bytes.NewReader(signed))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("certificate download failed: %d - %s", resp.StatusCode, string(body))
	}

	// Read certificate chain
	certPEM, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return certPEM, nil
}

// getNonce gets a fresh nonce from the ACME server
func (ac *ACMEClient) getNonce() (string, error) {
	ac.mu.RLock()
	nonceURL := ac.directory.NewNonce
	ac.mu.RUnlock()

	resp, err := ac.httpClient.Head(nonceURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	nonce := resp.Header.Get("Replay-Nonce")
	if nonce == "" {
		return "", errors.New("no nonce in response")
	}

	return nonce, nil
}

// getKeyAuthorization creates the key authorization for a challenge
func (ac *ACMEClient) getKeyAuthorization(token string) (string, error) {
	thumbprint, err := ac.getJWKThumbprint()
	if err != nil {
		return "", err
	}

	return token + "." + thumbprint, nil
}

// getJWKThumbprint computes the JWK thumbprint
func (ac *ACMEClient) getJWKThumbprint() (string, error) {
	pubKey := getPublicKey(ac.accountKey)
	jwk, err := publicKeyToJWK(pubKey)
	if err != nil {
		return "", err
	}

	// Serialize JWK in canonical form
	jwkJSON, err := json.Marshal(jwk)
	if err != nil {
		return "", err
	}

	// Compute SHA-256 hash
	hash := sha256.Sum256(jwkJSON)

	// Base64url encode
	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}

// signRequest signs an ACME request with JWS
func (ac *ACMEClient) signRequest(url, kid, nonce string, payload interface{}) ([]byte, error) {
	// Encode payload
	var payloadEncoded string
	if payload == "" {
		payloadEncoded = ""
	} else {
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		payloadEncoded = base64.RawURLEncoding.EncodeToString(payloadJSON)
	}

	// Create protected header
	protected := map[string]interface{}{
		"alg":   "ES256", // ECDSA with SHA-256
		"nonce": nonce,
		"url":   url,
	}

	if kid != "" {
		protected["kid"] = kid
	} else {
		pubKey := getPublicKey(ac.accountKey)
		jwk, err := publicKeyToJWK(pubKey)
		if err != nil {
			return nil, err
		}
		protected["jwk"] = jwk
	}

	protectedJSON, err := json.Marshal(protected)
	if err != nil {
		return nil, err
	}
	protectedEncoded := base64.RawURLEncoding.EncodeToString(protectedJSON)

	// Create signing input
	signingInput := protectedEncoded + "." + payloadEncoded

	// Sign
	signature, err := signData(ac.accountKey, []byte(signingInput))
	if err != nil {
		return nil, err
	}
	signatureEncoded := base64.RawURLEncoding.EncodeToString(signature)

	// Create JWS
	jws := map[string]string{
		"protected": protectedEncoded,
		"payload":   payloadEncoded,
		"signature": signatureEncoded,
	}

	return json.Marshal(jws)
}

// startChallengeServer starts the HTTP-01 challenge server on port 80
func (ac *ACMEClient) startChallengeServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/acme-challenge/", ac.handleChallenge)

	ac.challengeServer = &http.Server{
		Addr:    ":80",
		Handler: mux,
	}

	// Try to start server
	listener, err := net.Listen("tcp", ":80")
	if err != nil {
		// Port 80 might not be available (permission or already in use)
		// In production, you'd handle this more gracefully
		fmt.Printf("Warning: Could not start challenge server on port 80: %v\n", err)
		return nil
	}

	go func() {
		if err := ac.challengeServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Challenge server error: %v\n", err)
		}
	}()

	return nil
}

// handleChallenge handles HTTP-01 challenge requests
func (ac *ACMEClient) handleChallenge(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/.well-known/acme-challenge/")

	ac.challengeTokensMu.RLock()
	keyAuth, exists := ac.challengeTokens[token]
	ac.challengeTokensMu.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(keyAuth))
}

// Stop stops the ACME client
func (ac *ACMEClient) Stop() error {
	if ac.challengeServer != nil {
		return ac.challengeServer.Close()
	}
	return nil
}

// Helper functions

// publicKeyToJWK converts a public key to JWK format
func publicKeyToJWK(pubKey crypto.PublicKey) (map[string]interface{}, error) {
	switch key := pubKey.(type) {
	case *ecdsa.PublicKey:
		return map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(key.X.Bytes()),
			"y":   base64.RawURLEncoding.EncodeToString(key.Y.Bytes()),
		}, nil
	default:
		return nil, errors.New("unsupported key type")
	}
}

// signData signs data with the private key
func signData(privKey crypto.PrivateKey, data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)

	switch key := privKey.(type) {
	case *ecdsa.PrivateKey:
		r, s, err := ecdsa.Sign(rand.Reader, key, hash[:])
		if err != nil {
			return nil, err
		}
		// Encode as raw signature (r || s)
		signature := make([]byte, 64)
		r.FillBytes(signature[0:32])
		s.FillBytes(signature[32:64])
		return signature, nil
	default:
		return nil, errors.New("unsupported key type")
	}
}

// getPublicKey extracts the public key from a private key
func getPublicKey(privKey crypto.PrivateKey) crypto.PublicKey {
	switch key := privKey.(type) {
	case *ecdsa.PrivateKey:
		return &key.PublicKey
	case *rsa.PrivateKey:
		return &key.PublicKey
	default:
		return nil
	}
}
