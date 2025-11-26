# Shockwave TLS - Automatic Let's Encrypt Certificates

Complete TLS support with automatic certificate management via Let's Encrypt. Zero-configuration HTTPS with automatic renewal.

## Features

✅ **Automatic Certificate Acquisition** - Get certificates from Let's Encrypt automatically
✅ **Auto-Renewal** - Certificates renew 30 days before expiration
✅ **Staging Support** - Test with Let's Encrypt staging to avoid rate limits
✅ **Certificate Caching** - Certificates stored on disk and reused
✅ **HTTP-01 Challenge** - Automatic challenge handling on port 80
✅ **Secure Defaults** - TLS 1.2+, strong ciphers, perfect forward secrecy
✅ **HTTP/3 Support** - TLS 1.3 configuration for HTTP/3
✅ **Manual Certificates** - Also supports traditional cert/key files

## Quick Start

### Simplest Usage

```go
package main

import (
	"log"
	"net/http"

	"github.com/yourusername/shockwave/pkg/shockwave/tls"
)

func main() {
	// Get automatic HTTPS with one line
	tlsConfig, err := tls.QuickTLS("admin@example.com", "example.com", "www.example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Use with standard HTTP server
	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   myHandler,
	}

	log.Fatal(server.ListenAndServeTLS("", ""))
}
```

That's it! Your server now has automatic HTTPS with Let's Encrypt.

### Testing with Staging

When developing, use Let's Encrypt staging to avoid rate limits:

```go
tlsConfig, err := tls.QuickTLSStaging("admin@example.com", "example.com")
if err != nil {
	log.Fatal(err)
}
```

## Advanced Configuration

### Custom Settings

```go
config := tls.NewConfig().
	WithAutoCert("admin@example.com", "example.com").
	WithCertDir("/etc/letsencrypt/live").
	WithRenewBefore(30 * 24 * time.Hour).  // Renew 30 days before expiry
	WithCheckInterval(12 * time.Hour).      // Check every 12 hours
	WithMinTLSVersion(tls.VersionTLS12).
	WithALPN("h2", "http/1.1")

tlsConfig, err := config.Build()
if err != nil {
	log.Fatal(err)
}

server := &http.Server{
	Addr:      ":443",
	TLSConfig: tlsConfig,
	Handler:   myHandler,
}

log.Fatal(server.ListenAndServeTLS("", ""))
```

### Manual Certificates

If you already have certificates:

```go
tlsConfig, err := tls.ManualTLS("/path/to/cert.pem", "/path/to/key.pem")
if err != nil {
	log.Fatal(err)
}
```

### HTTP/3 Configuration

```go
config := tls.HTTP3Defaults().
	WithAutoCert("admin@example.com", "example.com")

tlsConfig, err := config.Build()
```

HTTP/3 defaults include:
- TLS 1.3 required
- h3 ALPN protocol
- Strong cipher suites

### Secure Defaults

For maximum security:

```go
config := tls.SecureDefaults().
	WithAutoCert("admin@example.com", "example.com")

tlsConfig, err := config.Build()
```

Secure defaults include:
- TLS 1.2+ only
- Strong ciphers with PFS (ECDHE)
- No weak algorithms
- Server cipher preference
- No renegotiation

## Certificate Management

### Monitoring Certificates

```go
config := tls.NewConfig().
	WithAutoCert("admin@example.com", "example.com")

tlsConfig, err := config.Build()
if err != nil {
	log.Fatal(err)
}

// Check certificate status
info := config.GetCertificateInfo()
for domain, cert := range info {
	fmt.Printf("%s: %d days until expiry\n", domain, cert.DaysUntilExpiry())
	fmt.Printf("  Issued: %v\n", cert.IssuedAt)
	fmt.Printf("  Expires: %v\n", cert.ExpiresAt)
	fmt.Printf("  Valid: %v\n", cert.IsValid())
}
```

### Manual Renewal

```go
// Force renewal of a certificate
err := config.RenewCertificate("example.com")
if err != nil {
	log.Printf("Renewal failed: %v", err)
}
```

### Graceful Shutdown

```go
// Stop the certificate manager when shutting down
defer config.Stop()
```

## Configuration Options

### Auto-Cert Options

| Option | Description | Default |
|--------|-------------|---------|
| `Email` | Email for Let's Encrypt account | Required |
| `Domains` | List of domains to manage | Required |
| `CertDir` | Directory to store certificates | `./certs` |
| `Staging` | Use Let's Encrypt staging | `false` |
| `RenewBefore` | Renew certificates before expiry | `30 days` |
| `CheckInterval` | How often to check for renewals | `12 hours` |

### TLS Options

| Option | Description | Default |
|--------|-------------|---------|
| `MinVersion` | Minimum TLS version | TLS 1.2 |
| `MaxVersion` | Maximum TLS version | TLS 1.3 |
| `CipherSuites` | Allowed cipher suites | Strong ciphers |
| `PreferServerCiphers` | Server chooses cipher | `true` |
| `NextProtos` | ALPN protocols | `["h2", "http/1.1"]` |
| `ClientAuth` | Client certificate auth | `NoClientCert` |

## Production Deployment

### Requirements

1. **Port 80 Access**: Required for HTTP-01 challenge
2. **DNS Configuration**: Domains must point to your server
3. **File System Access**: Certificates stored in `CertDir`

### Best Practices

```go
config := tls.NewConfig().
	WithAutoCert("admin@example.com", "example.com", "www.example.com").
	WithCertDir("/var/lib/letsencrypt").  // Persistent storage
	WithRenewBefore(30 * 24 * time.Hour). // 30 days before expiry
	WithCheckInterval(12 * time.Hour).     // Check twice daily
	WithMinTLSVersion(tls.VersionTLS12).  // TLS 1.2+
	WithALPN("h2", "http/1.1")            // HTTP/2 support

tlsConfig, err := config.Build()
if err != nil {
	log.Fatal(err)
}

// Ensure graceful shutdown
defer config.Stop()

server := &http.Server{
	Addr:         ":443",
	TLSConfig:    tlsConfig,
	Handler:      myHandler,
	ReadTimeout:  10 * time.Second,
	WriteTimeout: 10 * time.Second,
	IdleTimeout:  120 * time.Second,
}

// Graceful shutdown
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go func() {
	<-sigChan
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	config.Stop()
}()

log.Fatal(server.ListenAndServeTLS("", ""))
```

### Rate Limits

Let's Encrypt has rate limits:
- **Production**: 50 certificates per week per domain
- **Staging**: Much higher limits

**Always test with staging first!**

```go
// Development
tlsConfig, _ := tls.QuickTLSStaging("admin@example.com", "example.com")

// Production
tlsConfig, _ := tls.QuickTLS("admin@example.com", "example.com")
```

## Certificate Storage

Certificates are stored in the following structure:

```
./certs/
├── account.key          # ACME account key
├── example.com.crt      # Certificate chain
├── example.com.key      # Private key
├── www.example.com.crt
└── www.example.com.key
```

All files are created with `0600` permissions (owner read/write only).

## How It Works

### Certificate Acquisition Flow

1. **Initial Request**: Server starts, checks for existing certificates
2. **ACME Registration**: Registers account with Let's Encrypt (if needed)
3. **Domain Validation**:
   - Creates HTTP-01 challenge response on port 80
   - Let's Encrypt verifies domain ownership
4. **Certificate Issuance**: Let's Encrypt issues certificate
5. **Storage**: Certificate and key saved to disk
6. **Caching**: Certificate loaded into memory for fast access

### Auto-Renewal Flow

1. **Background Monitor**: Checks certificates every 12 hours (configurable)
2. **Expiry Check**: Identifies certificates expiring within 30 days
3. **Automatic Renewal**: Requests new certificate from Let's Encrypt
4. **Seamless Update**: New certificate loaded without downtime
5. **Storage Update**: Updated certificate saved to disk

## Security Features

### Strong Defaults

All configurations use secure defaults:

✅ **TLS 1.2+** - No TLS 1.0 or 1.1
✅ **Perfect Forward Secrecy** - ECDHE cipher suites only
✅ **Strong Encryption** - AES-256-GCM, AES-128-GCM, ChaCha20-Poly1305
✅ **No Weak Algorithms** - No RC4, no 3DES, no MD5
✅ **Server Cipher Preference** - Server chooses strongest cipher
✅ **No Renegotiation** - Prevents renegotiation attacks

### Certificate Validation

- **Domain Validation**: HTTP-01 challenge verifies domain ownership
- **Certificate Chain**: Full chain validation
- **Expiry Checking**: Automatic renewal before expiration
- **Secure Storage**: Private keys stored with 0600 permissions

## Troubleshooting

### Port 80 Required

```
Error: Could not start challenge server on port 80
```

**Solution**: Ensure port 80 is available. Let's Encrypt requires HTTP-01 challenge on port 80.

```bash
# Check if port 80 is in use
sudo lsof -i :80

# Grant permission to bind to port 80 (Linux)
sudo setcap CAP_NET_BIND_SERVICE=+eip /path/to/your/binary
```

### DNS Not Configured

```
Error: authorization failed
```

**Solution**: Ensure your domain's DNS points to your server's IP address.

```bash
# Verify DNS
dig example.com
nslookup example.com
```

### Rate Limit Exceeded

```
Error: too many certificates already issued
```

**Solution**: Use staging environment for development:

```go
tlsConfig, _ := tls.QuickTLSStaging("admin@example.com", "example.com")
```

### Certificate Not Renewing

Check renewal settings:

```go
config := tls.NewConfig().
	WithAutoCert("admin@example.com", "example.com").
	WithRenewBefore(30 * 24 * time.Hour).  // Increase if needed
	WithCheckInterval(6 * time.Hour)        // Check more frequently
```

## Examples

### Multi-Domain Certificate

```go
tlsConfig, err := tls.QuickTLS(
	"admin@example.com",
	"example.com",
	"www.example.com",
	"api.example.com",
)
```

### Custom Cipher Suites

```go
config := tls.NewConfig().
	WithAutoCert("admin@example.com", "example.com").
	WithCipherSuites([]uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	})
```

### Client Certificate Authentication

```go
config := tls.NewConfig().
	WithAutoCert("admin@example.com", "example.com").
	WithClientAuth(tls.RequireAndVerifyClientCert)
```

### HTTP Redirect to HTTPS

```go
// Start HTTP server that redirects to HTTPS
go http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	target := "https://" + r.Host + r.URL.Path
	if len(r.URL.RawQuery) > 0 {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusPermanentRedirect)
}))

// HTTPS server with auto-cert
tlsConfig, _ := tls.QuickTLS("admin@example.com", "example.com")
server := &http.Server{
	Addr:      ":443",
	TLSConfig: tlsConfig,
	Handler:   myHandler,
}
server.ListenAndServeTLS("", "")
```

## Performance

- **Certificate Lookup**: O(1) in-memory cache
- **Key Generation**: ECDSA P-256 (fast) by default
- **Challenge Response**: Minimal overhead on port 80
- **Renewal**: Non-blocking background process

## Compatibility

- **Go**: 1.18+
- **Let's Encrypt**: ACME v2 (RFC 8555)
- **TLS**: 1.2, 1.3
- **HTTP**: HTTP/1.1, HTTP/2, HTTP/3

## License

Part of the Shockwave HTTP library.

## Contributing

See main Shockwave documentation.
