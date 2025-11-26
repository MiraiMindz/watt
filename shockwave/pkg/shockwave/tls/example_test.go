package tls_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/shockwave/pkg/shockwave/tls"
	stdtls "crypto/tls"
)

// Example_quickTLS demonstrates the simplest way to get automatic HTTPS
func Example_quickTLS() {
	// One line to get automatic HTTPS with Let's Encrypt
	tlsConfig, err := tls.QuickTLS("admin@example.com", "example.com", "www.example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Use with standard HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, HTTPS!")
	})

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	log.Println("Server starting on :443 with automatic HTTPS")
	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Example_staging demonstrates using Let's Encrypt staging for testing
func Example_staging() {
	// Use staging environment to avoid rate limits during development
	tlsConfig, err := tls.QuickTLSStaging("admin@example.com", "test.example.com")
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	log.Println("Server starting with Let's Encrypt staging")
	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Example_advanced demonstrates advanced configuration
func Example_advanced() {
	config := tls.NewConfig().
		WithAutoCert("admin@example.com", "example.com").
		WithCertDir("/etc/letsencrypt/live").
		WithRenewBefore(30 * 24 * time.Hour).
		WithCheckInterval(12 * time.Hour).
		WithMinTLSVersion(stdtls.VersionTLS12).
		WithALPN("h2", "http/1.1")

	tlsConfig, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}

	// Ensure graceful shutdown
	defer config.Stop()

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Example_manualCert demonstrates using manual certificate files
func Example_manualCert() {
	tlsConfig, err := tls.ManualTLS("/path/to/cert.pem", "/path/to/key.pem")
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Example_http3 demonstrates HTTP/3 configuration
func Example_http3() {
	config := tls.HTTP3Defaults().
		WithAutoCert("admin@example.com", "example.com")

	tlsConfig, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}

	// HTTP/3 server would use this config
	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	log.Println("HTTP/3 server starting")
	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Example_monitoring demonstrates certificate monitoring
func Example_monitoring() {
	config := tls.NewConfig().
		WithAutoCert("admin@example.com", "example.com")

	tlsConfig, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}

	// Start monitoring in background
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			info := config.GetCertificateInfo()
			for domain, cert := range info {
				days := cert.DaysUntilExpiry()
				log.Printf("%s: %d days until expiry", domain, days)

				if days < 7 {
					log.Printf("WARNING: Certificate for %s expires soon!", domain)
				}
			}
		}
	}()

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Example_gracefulShutdown demonstrates graceful shutdown with certificate cleanup
func Example_gracefulShutdown() {
	config := tls.NewConfig().
		WithAutoCert("admin@example.com", "example.com")

	tlsConfig, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, HTTPS!")
	})

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   http.DefaultServeMux,
	}

	// Start server in background
	go func() {
		log.Println("Server starting on :443")
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")

	// Shutdown server with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Stop certificate manager
	config.Stop()

	log.Println("Server stopped")
}

// Example_httpRedirect demonstrates redirecting HTTP to HTTPS
func Example_httpRedirect() {
	// Start HTTP redirect server
	go func() {
		http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + r.Host + r.URL.Path
			if len(r.URL.RawQuery) > 0 {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusPermanentRedirect)
		}))
	}()

	// HTTPS server with auto-cert
	tlsConfig, err := tls.QuickTLS("admin@example.com", "example.com")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Secure HTTPS connection!")
	})

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Example_multiDomain demonstrates managing multiple domains
func Example_multiDomain() {
	tlsConfig, err := tls.QuickTLS(
		"admin@example.com",
		"example.com",
		"www.example.com",
		"api.example.com",
		"blog.example.com",
	)
	if err != nil {
		log.Fatal(err)
	}

	// All domains will get certificates automatically
	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Host: %s", r.Host)
		}),
	}

	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Example_secureDefaults demonstrates maximum security settings
func Example_secureDefaults() {
	config := tls.SecureDefaults().
		WithAutoCert("admin@example.com", "example.com")

	tlsConfig, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}

	// This configuration uses:
	// - TLS 1.2+ only
	// - Strong ciphers with PFS
	// - Server cipher preference
	// - No renegotiation

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Example_customCiphers demonstrates custom cipher suite selection
func Example_customCiphers() {
	config := tls.NewConfig().
		WithAutoCert("admin@example.com", "example.com").
		WithCipherSuites([]uint16{
			stdtls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			stdtls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			stdtls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			stdtls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		})

	tlsConfig, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Example_manualRenewal demonstrates manually triggering certificate renewal
func Example_manualRenewal() {
	config := tls.NewConfig().
		WithAutoCert("admin@example.com", "example.com")

	tlsConfig, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}

	// Set up HTTP endpoint to trigger renewal
	http.HandleFunc("/admin/renew-cert", func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		if domain == "" {
			http.Error(w, "domain parameter required", http.StatusBadRequest)
			return
		}

		err := config.RenewCertificate(domain)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Certificate renewal queued for %s", domain)
	})

	// Status endpoint
	http.HandleFunc("/admin/cert-status", func(w http.ResponseWriter, r *http.Request) {
		info := config.GetCertificateInfo()
		for domain, cert := range info {
			fmt.Fprintf(w, "%s:\n", domain)
			fmt.Fprintf(w, "  Days until expiry: %d\n", cert.DaysUntilExpiry())
			fmt.Fprintf(w, "  Valid: %v\n", cert.IsValid())
			fmt.Fprintf(w, "  Issued: %v\n", cert.IssuedAt)
			fmt.Fprintf(w, "  Expires: %v\n", cert.ExpiresAt)
			fmt.Fprintf(w, "\n")
		}
	})

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	log.Fatal(server.ListenAndServeTLS("", ""))
}
