// Package http11 implements a high-performance HTTP/1.1 engine with zero-allocation parsing.
package http11

// HTTP Method IDs for O(1) switching
// These numeric IDs enable fast method identification without string comparisons
const (
	MethodUnknown uint8 = 0
	MethodGET     uint8 = 1
	MethodPOST    uint8 = 2
	MethodPUT     uint8 = 3
	MethodDELETE  uint8 = 4
	MethodPATCH   uint8 = 5
	MethodHEAD    uint8 = 6
	MethodOPTIONS uint8 = 7
	MethodCONNECT uint8 = 8
	MethodTRACE   uint8 = 9
)

// HTTP Methods - Byte slices for parsing (zero allocations)
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

// HTTP Status Lines - Pre-compiled with CRLF for zero-allocation writes
// Covers 95% of HTTP responses - all common status codes
var (
	// 1xx Informational
	status100Bytes = []byte("HTTP/1.1 100 Continue\r\n")
	status101Bytes = []byte("HTTP/1.1 101 Switching Protocols\r\n")

	// 2xx Success
	status200Bytes = []byte("HTTP/1.1 200 OK\r\n")
	status201Bytes = []byte("HTTP/1.1 201 Created\r\n")
	status202Bytes = []byte("HTTP/1.1 202 Accepted\r\n")
	status203Bytes = []byte("HTTP/1.1 203 Non-Authoritative Information\r\n")
	status204Bytes = []byte("HTTP/1.1 204 No Content\r\n")
	status205Bytes = []byte("HTTP/1.1 205 Reset Content\r\n")
	status206Bytes = []byte("HTTP/1.1 206 Partial Content\r\n")

	// 3xx Redirection
	status300Bytes = []byte("HTTP/1.1 300 Multiple Choices\r\n")
	status301Bytes = []byte("HTTP/1.1 301 Moved Permanently\r\n")
	status302Bytes = []byte("HTTP/1.1 302 Found\r\n")
	status303Bytes = []byte("HTTP/1.1 303 See Other\r\n")
	status304Bytes = []byte("HTTP/1.1 304 Not Modified\r\n")
	status307Bytes = []byte("HTTP/1.1 307 Temporary Redirect\r\n")
	status308Bytes = []byte("HTTP/1.1 308 Permanent Redirect\r\n")

	// 4xx Client Error
	status400Bytes = []byte("HTTP/1.1 400 Bad Request\r\n")
	status401Bytes = []byte("HTTP/1.1 401 Unauthorized\r\n")
	status403Bytes = []byte("HTTP/1.1 403 Forbidden\r\n")
	status404Bytes = []byte("HTTP/1.1 404 Not Found\r\n")
	status405Bytes = []byte("HTTP/1.1 405 Method Not Allowed\r\n")
	status406Bytes = []byte("HTTP/1.1 406 Not Acceptable\r\n")
	status408Bytes = []byte("HTTP/1.1 408 Request Timeout\r\n")
	status409Bytes = []byte("HTTP/1.1 409 Conflict\r\n")
	status410Bytes = []byte("HTTP/1.1 410 Gone\r\n")
	status411Bytes = []byte("HTTP/1.1 411 Length Required\r\n")
	status412Bytes = []byte("HTTP/1.1 412 Precondition Failed\r\n")
	status413Bytes = []byte("HTTP/1.1 413 Payload Too Large\r\n")
	status414Bytes = []byte("HTTP/1.1 414 URI Too Long\r\n")
	status415Bytes = []byte("HTTP/1.1 415 Unsupported Media Type\r\n")
	status429Bytes = []byte("HTTP/1.1 429 Too Many Requests\r\n")

	// 5xx Server Error
	status500Bytes = []byte("HTTP/1.1 500 Internal Server Error\r\n")
	status501Bytes = []byte("HTTP/1.1 501 Not Implemented\r\n")
	status502Bytes = []byte("HTTP/1.1 502 Bad Gateway\r\n")
	status503Bytes = []byte("HTTP/1.1 503 Service Unavailable\r\n")
	status504Bytes = []byte("HTTP/1.1 504 Gateway Timeout\r\n")
)

// HTTP Status Lines - String versions
const (
	status200String = "HTTP/1.1 200 OK\r\n"
	status201String = "HTTP/1.1 201 Created\r\n"
	status204String = "HTTP/1.1 204 No Content\r\n"
	status301String = "HTTP/1.1 301 Moved Permanently\r\n"
	status302String = "HTTP/1.1 302 Found\r\n"
	status304String = "HTTP/1.1 304 Not Modified\r\n"
	status400String = "HTTP/1.1 400 Bad Request\r\n"
	status401String = "HTTP/1.1 401 Unauthorized\r\n"
	status403String = "HTTP/1.1 403 Forbidden\r\n"
	status404String = "HTTP/1.1 404 Not Found\r\n"
	status500String = "HTTP/1.1 500 Internal Server Error\r\n"
	status502String = "HTTP/1.1 502 Bad Gateway\r\n"
	status503String = "HTTP/1.1 503 Service Unavailable\r\n"
)

// Common HTTP Headers - Byte slices for zero-allocation parsing
var (
	headerContentLength    = []byte("Content-Length")
	headerContentType      = []byte("Content-Type")
	headerConnection       = []byte("Connection")
	headerKeepAlive        = []byte("keep-alive")
	headerClose            = []byte("close")
	headerTransferEncoding = []byte("Transfer-Encoding")
	headerChunked          = []byte("chunked")
	headerHost             = []byte("Host")
	headerUserAgent        = []byte("User-Agent")
	headerAccept           = []byte("Accept")
	headerAcceptEncoding   = []byte("Accept-Encoding")
	headerAcceptLanguage   = []byte("Accept-Language")
	headerCacheControl     = []byte("Cache-Control")
	headerCookie           = []byte("Cookie")
	headerSetCookie        = []byte("Set-Cookie")
	headerAuthorization    = []byte("Authorization")
	headerLocation         = []byte("Location")
	headerServer           = []byte("Server")
	headerDate             = []byte("Date")
	headerExpires          = []byte("Expires")
	headerETag             = []byte("ETag")
	headerLastModified     = []byte("Last-Modified")
	headerIfModifiedSince  = []byte("If-Modified-Since")
	headerIfNoneMatch      = []byte("If-None-Match")
	headerRange            = []byte("Range")
	headerContentRange     = []byte("Content-Range")
	headerUpgrade          = []byte("Upgrade")
	headerOrigin           = []byte("Origin")
	headerReferer          = []byte("Referer")
)

// Common Content-Type values - Pre-compiled for zero allocations
// P2 FIX #7: Extended content-type constants for modern web applications
var (
	// Text & Documents
	contentTypeJSON       = []byte("application/json")
	contentTypeJSONUTF8   = []byte("application/json; charset=utf-8")
	contentTypeHTML       = []byte("text/html; charset=utf-8")
	contentTypePlain      = []byte("text/plain; charset=utf-8")
	contentTypeXML        = []byte("application/xml")
	contentTypePDF        = []byte("application/pdf")
	contentTypeMarkdown   = []byte("text/markdown; charset=utf-8")

	// Web Application Formats
	contentTypeForm       = []byte("application/x-www-form-urlencoded")
	contentTypeMultipart  = []byte("multipart/form-data")
	contentTypeJavaScript = []byte("application/javascript")
	contentTypeCSS        = []byte("text/css")
	contentTypeWasm       = []byte("application/wasm")

	// API & Data Exchange
	contentTypeJSONAPI    = []byte("application/vnd.api+json")
	contentTypeJSONLD     = []byte("application/ld+json")
	contentTypeProtobuf   = []byte("application/x-protobuf")
	contentTypeMsgPack    = []byte("application/msgpack")
	contentTypeYAML       = []byte("application/x-yaml")
	contentTypeTOML       = []byte("application/toml")

	// Images - Raster
	contentTypePNG        = []byte("image/png")
	contentTypeJPEG       = []byte("image/jpeg")
	contentTypeGIF        = []byte("image/gif")
	contentTypeWebP       = []byte("image/webp")
	contentTypeAVIF       = []byte("image/avif")
	contentTypeBMP        = []byte("image/bmp")
	contentTypeICO        = []byte("image/x-icon")

	// Images - Vector
	contentTypeSVG        = []byte("image/svg+xml")

	// Audio
	contentTypeMP3        = []byte("audio/mpeg")
	contentTypeOGG        = []byte("audio/ogg")
	contentTypeWAV        = []byte("audio/wav")
	contentTypeAAC        = []byte("audio/aac")
	contentTypeFLAC       = []byte("audio/flac")
	contentTypeOpus       = []byte("audio/opus")

	// Video
	contentTypeMP4        = []byte("video/mp4")
	contentTypeWebM       = []byte("video/webm")
	contentTypeOGV        = []byte("video/ogg")
	contentTypeMOV        = []byte("video/quicktime")
	contentTypeAVI        = []byte("video/x-msvideo")

	// Fonts
	contentTypeWOFF       = []byte("font/woff")
	contentTypeWOFF2      = []byte("font/woff2")
	contentTypeTTF        = []byte("font/ttf")
	contentTypeOTF        = []byte("font/otf")
	contentTypeEOT        = []byte("application/vnd.ms-fontobject")

	// Archives
	contentTypeZIP        = []byte("application/zip")
	contentTypeGZIP       = []byte("application/gzip")
	contentTypeTAR        = []byte("application/x-tar")
	contentTypeBZIP2      = []byte("application/x-bzip2")
	contentType7Z         = []byte("application/x-7z-compressed")

	// Streaming
	contentTypeEventStream = []byte("text/event-stream")
	contentTypeM3U8       = []byte("application/vnd.apple.mpegurl")
	contentTypeMPD        = []byte("application/dash+xml")

	// Binary
	contentTypeOctetStream = []byte("application/octet-stream")
)

// Protocol constants
var (
	http11Bytes = []byte("HTTP/1.1")
	http10Bytes = []byte("HTTP/1.0")
	crlfBytes   = []byte("\r\n")
	colonSpace  = []byte(": ")
	http11Proto = "HTTP/1.1"
)

// HTTP/1.1 protocol version
const (
	ProtoHTTP11Major = 1
	ProtoHTTP11Minor = 1
)

// Header and request limits (per RFC 7230 and security best practices)
const (
	// MaxHeaders is the maximum number of headers we can store inline without heap allocation
	MaxHeaders = 32

	// MaxHeaderName is the maximum length of a header name
	MaxHeaderName = 64

	// MaxHeaderValue is the maximum length of a header value for inline storage
	// Values larger than this will use overflow storage (heap allocation)
	// 128 bytes covers 99% of headers; large values like cookies use overflow
	MaxHeaderValue = 128

	// MaxRequestLineSize is the maximum size of the request line (method + path + protocol)
	MaxRequestLineSize = 8192

	// P0 FIX #5: Excessive URI Length DoS Protection
	// MaxURILength is the maximum length of the Request-URI
	// RFC 7230 recommends at least 8000 octets; we use 8KB to prevent DoS
	// Extremely long URIs can cause memory exhaustion and slowloris-style attacks
	MaxURILength = 8192

	// MaxHeadersSize is the maximum total size of all headers
	MaxHeadersSize = 8192
)

// Common JSON responses - Pre-compiled for zero allocations
var (
	jsonOK               = []byte(`{"status":"ok"}`)
	jsonError            = []byte(`{"status":"error"}`)
	jsonNotFound         = []byte(`{"status":"error","message":"not found"}`)
	jsonBadRequest       = []byte(`{"status":"error","message":"bad request"}`)
	jsonInternalError    = []byte(`{"status":"error","message":"internal server error"}`)
	jsonUnauthorized     = []byte(`{"status":"error","message":"unauthorized"}`)
	jsonForbidden        = []byte(`{"status":"error","message":"forbidden"}`)
	jsonMethodNotAllowed = []byte(`{"status":"error","message":"method not allowed"}`)
)
