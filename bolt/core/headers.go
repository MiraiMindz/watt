package core

// Pre-compiled header constants to avoid string allocations on every request.
//
// Using byte slices instead of strings eliminates allocations when setting headers.
// These constants are shared across all requests and never modified.
//
// Performance impact:
//   - Before: SetHeader("Content-Type", "application/json") = 2 allocs
//   - After:  setContentTypeJSON() = 0 allocs
//
// Savings: 2-3 allocations per request for common headers.

// Header names (byte slice constants)
var (
	headerContentType            = []byte("Content-Type")
	headerContentLength          = []byte("Content-Length")
	headerServer                 = []byte("Server")
	headerDate                   = []byte("Date")
	headerConnection             = []byte("Connection")
	headerCacheControl           = []byte("Cache-Control")
	headerAccessControlAllowOrigin = []byte("Access-Control-Allow-Origin")
)

// Content-Type values (byte slice constants)
var (
	contentTypeJSON       = []byte("application/json")
	contentTypeJSONUTF8   = []byte("application/json; charset=utf-8")
	contentTypeText       = []byte("text/plain; charset=utf-8")
	contentTypeHTML       = []byte("text/html; charset=utf-8")
	contentTypeXML        = []byte("application/xml; charset=utf-8")
	contentTypeFormData   = []byte("application/x-www-form-urlencoded")
	contentTypeMultipart  = []byte("multipart/form-data")
	contentTypeOctetStream = []byte("application/octet-stream")
)

// ✅ PHASE 1.3: Pre-allocated header value slices (bypass net/textproto allocation)
// These are shared, read-only slices that can be assigned directly to http.Header maps
// without going through Header().Set() which triggers expensive canonicalization.
//
// Performance: Eliminates ~60-80ns of net/textproto overhead per request
var (
	contentTypeJSONSlice   = []string{"application/json"}
	contentTypeJSONUTF8Slice = []string{"application/json; charset=utf-8"}
	contentTypeTextSlice   = []string{"text/plain; charset=utf-8"}
	contentTypeHTMLSlice   = []string{"text/html; charset=utf-8"}
	contentTypeXMLSlice    = []string{"application/xml; charset=utf-8"}
	serverBoltSlice        = []string{"Bolt"}
	cacheNoCacheSlice      = []string{"no-cache, no-store, must-revalidate"}
	corsAllowAllSlice      = []string{"*"}
)

// Other common header values
var (
	serverBolt       = []byte("Bolt")
	connectionKeepAlive = []byte("keep-alive")
	connectionClose     = []byte("close")
	cacheNoCache        = []byte("no-cache, no-store, must-revalidate")
	corsAllowAll        = []byte("*")
)

// setContentTypeJSON sets Content-Type to application/json (zero-allocation).
//
// ✅ PHASE 1.1: Bypass net/textproto overhead (~60-80ns gain)
// Writes directly to header map instead of using Header().Set() which calls:
// - CanonicalMIMEHeaderKey (~230ms in CPU profile)
// - validHeaderFieldByte (~180ms in CPU profile)
//
// Performance: 0 allocs/op, ~60-80ns faster than Header().Set()
//
//go:inline
func (c *Context) setContentTypeJSON() {
	// ✅ FAST PATH: Direct map write for net/http (bypasses net/textproto validation)
	if c.httpRes != nil {
		// Use canonical key "Content-Type" and pre-allocated slice
		c.httpRes.Header()["Content-Type"] = contentTypeJSONSlice
		return
	}

	// Shockwave (already zero-copy, no overhead)
	if c.shockwaveRes != nil {
		_ = c.shockwaveRes.Header().Set(headerContentType, contentTypeJSON)
		return
	}

	// Test mode
	if c.testResHeaders == nil {
		c.testResHeaders = make(map[string]string, 4)
	}
	c.testResHeaders["Content-Type"] = "application/json"
}

// setContentTypeText sets Content-Type to text/plain; charset=utf-8 (zero-allocation).
//
// ✅ PHASE 1.1: Bypass net/textproto overhead
//
//go:inline
func (c *Context) setContentTypeText() {
	if c.httpRes != nil {
		c.httpRes.Header()["Content-Type"] = contentTypeTextSlice
		return
	}
	if c.shockwaveRes != nil {
		_ = c.shockwaveRes.Header().Set(headerContentType, contentTypeText)
		return
	}
	if c.testResHeaders == nil {
		c.testResHeaders = make(map[string]string, 4)
	}
	c.testResHeaders["Content-Type"] = "text/plain; charset=utf-8"
}

// setContentTypeHTML sets Content-Type to text/html; charset=utf-8 (zero-allocation).
//
// ✅ PHASE 1.1: Bypass net/textproto overhead
//
//go:inline
func (c *Context) setContentTypeHTML() {
	if c.httpRes != nil {
		c.httpRes.Header()["Content-Type"] = contentTypeHTMLSlice
		return
	}
	if c.shockwaveRes != nil {
		_ = c.shockwaveRes.Header().Set(headerContentType, contentTypeHTML)
		return
	}
	if c.testResHeaders == nil {
		c.testResHeaders = make(map[string]string, 4)
	}
	c.testResHeaders["Content-Type"] = "text/html; charset=utf-8"
}

// setContentTypeXML sets Content-Type to application/xml; charset=utf-8 (zero-allocation).
//
// Performance: 0 allocs/op (vs 2 allocs with SetHeader)
func (c *Context) setContentTypeXML() {
	c.SetHeaderBytes(headerContentType, contentTypeXML)
}

// SetServerHeader sets the Server header to "Bolt" (zero-allocation).
//
// Performance: 0 allocs/op (vs 2 allocs with SetHeader)
func (c *Context) SetServerHeader() {
	c.SetHeaderBytes(headerServer, serverBolt)
}

// SetNoCacheHeaders sets cache-control headers to prevent caching (zero-allocation).
//
// Performance: 0 allocs/op (vs 2 allocs with SetHeader)
func (c *Context) SetNoCacheHeaders() {
	c.SetHeaderBytes(headerCacheControl, cacheNoCache)
}

// SetCORSAllowAll sets Access-Control-Allow-Origin to * (zero-allocation).
//
// Performance: 0 allocs/op (vs 2 allocs with SetHeader)
func (c *Context) SetCORSAllowAll() {
	c.SetHeaderBytes(headerAccessControlAllowOrigin, corsAllowAll)
}
