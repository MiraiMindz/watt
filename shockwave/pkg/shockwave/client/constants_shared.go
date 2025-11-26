package client

// Pre-compiled HTTP constants for zero-allocation client operations.
// Both byte slice and string versions are provided for different use cases.

// HTTP Methods - Byte slices for writing (zero allocations)
var (
	methodGETBytes     = []byte("GET")
	methodPOSTBytes    = []byte("POST")
	methodPUTBytes     = []byte("PUT")
	methodDELETEBytes  = []byte("DELETE")
	methodPATCHBytes   = []byte("PATCH")
	methodHEADBytes    = []byte("HEAD")
	methodOPTIONSBytes = []byte("OPTIONS")
	methodCONNECTBytes = []byte("CONNECT")
	methodTRACEBytes   = []byte("TRACE")
)

// HTTP Methods - Strings for comparison (zero allocations)
const (
	methodGETString     = "GET"
	methodPOSTString    = "POST"
	methodPUTString     = "PUT"
	methodDELETEString  = "DELETE"
	methodPATCHString   = "PATCH"
	methodHEADString    = "HEAD"
	methodOPTIONSString = "OPTIONS"
	methodCONNECTString = "CONNECT"
	methodTRACEString   = "TRACE"
)

// HTTP Method IDs for O(1) switching
const (
	methodIDUnknown uint8 = 0
	methodIDGET     uint8 = 1
	methodIDPOST    uint8 = 2
	methodIDPUT     uint8 = 3
	methodIDDELETE  uint8 = 4
	methodIDPATCH   uint8 = 5
	methodIDHEAD    uint8 = 6
	methodIDOPTIONS uint8 = 7
	methodIDCONNECT uint8 = 8
	methodIDTRACE   uint8 = 9
)

// Protocol constants
var (
	http11Bytes    = []byte("HTTP/1.1")
	http10Bytes    = []byte("HTTP/1.0")
	crlfBytes      = []byte("\r\n")
	colonSpaceBytes = []byte(": ")
	spaceBytes     = []byte(" ")
	questionBytes  = []byte("?")
)

const (
	http11String = "HTTP/1.1"
	http10String = "HTTP/1.0"
	crlfString   = "\r\n"
)

// Common HTTP Headers - Byte slices for zero-allocation operations
var (
	headerHost             = []byte("Host")
	headerUserAgent        = []byte("User-Agent")
	headerConnection       = []byte("Connection")
	headerContentLength    = []byte("Content-Length")
	headerContentType      = []byte("Content-Type")
	headerTransferEncoding = []byte("Transfer-Encoding")
	headerChunked          = []byte("chunked")
	headerKeepAlive        = []byte("keep-alive")
	headerClose            = []byte("close")
	headerAccept           = []byte("Accept")
	headerAcceptEncoding   = []byte("Accept-Encoding")
	headerAcceptLanguage   = []byte("Accept-Language")
	headerAuthorization    = []byte("Authorization")
	headerCookie           = []byte("Cookie")
	headerReferer          = []byte("Referer")
	headerOrigin           = []byte("Origin")
	headerRange            = []byte("Range")
	headerIfModifiedSince  = []byte("If-Modified-Since")
	headerIfNoneMatch      = []byte("If-None-Match")
)

// Common Content-Type values - Pre-compiled for zero allocations
var (
	contentTypeJSON       = []byte("application/json")
	contentTypeJSONUTF8   = []byte("application/json; charset=utf-8")
	contentTypeForm       = []byte("application/x-www-form-urlencoded")
	contentTypeMultipart  = []byte("multipart/form-data")
	contentTypePlain      = []byte("text/plain")
	contentTypeHTML       = []byte("text/html")
	contentTypeXML        = []byte("application/xml")
	contentTypeOctetStream = []byte("application/octet-stream")
)

// MaxHeaders, MaxHeaderName, MaxHeaderValue, and MaxStatusLine are defined in
// build-tag specific files (constants_lowmem.go, constants_highperf.go, etc.)
// to allow memory/performance trade-offs based on use case.

// Buffer sizes
const (
	// DefaultBufferSize for read/write operations
	DefaultBufferSize = 4096

	// LargeBufferSize for request/response parsing
	LargeBufferSize = 16384
)

// Common User-Agent strings
var (
	defaultUserAgent = []byte("Shockwave-Client/1.0")
)

// methodToBytes converts a method string to its pre-compiled byte slice
// Returns pre-compiled constant for zero allocations
func methodToBytes(method string) []byte {
	switch method {
	case methodGETString:
		return methodGETBytes
	case methodPOSTString:
		return methodPOSTBytes
	case methodPUTString:
		return methodPUTBytes
	case methodDELETEString:
		return methodDELETEBytes
	case methodPATCHString:
		return methodPATCHBytes
	case methodHEADString:
		return methodHEADBytes
	case methodOPTIONSString:
		return methodOPTIONSBytes
	case methodCONNECTString:
		return methodCONNECTBytes
	case methodTRACEString:
		return methodTRACEBytes
	default:
		// Unknown method - caller will handle allocation
		return nil
	}
}

// methodToID converts a method string to its ID for O(1) switching
func methodToID(method string) uint8 {
	switch method {
	case methodGETString:
		return methodIDGET
	case methodPOSTString:
		return methodIDPOST
	case methodPUTString:
		return methodIDPUT
	case methodDELETEString:
		return methodIDDELETE
	case methodPATCHString:
		return methodIDPATCH
	case methodHEADString:
		return methodIDHEAD
	case methodOPTIONSString:
		return methodIDOPTIONS
	case methodCONNECTString:
		return methodIDCONNECT
	case methodTRACEString:
		return methodIDTRACE
	default:
		return methodIDUnknown
	}
}

// bytesEqual compares two byte slices without allocating
// This is faster than bytes.Equal for small slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// bytesEqualCaseInsensitive compares byte slices case-insensitively
// Optimized for ASCII (HTTP headers are ASCII-only)
func bytesEqualCaseInsensitive(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca := a[i]
		cb := b[i]

		// Convert to lowercase (ASCII only)
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}

		if ca != cb {
			return false
		}
	}
	return true
}
