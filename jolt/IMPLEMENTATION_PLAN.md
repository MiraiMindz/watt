# Jolt Implementation Plan

## Overview
Jolt is Watt's high-performance partial page update system inspired by htmx, enabling AJAX-like interactions without writing JavaScript. Like an electrical jolt sends a sharp burst of energy, Jolt sends small, targeted bursts of HTML to update specific page regions.

## Core Principles
- **Hypermedia-Driven**: HTML over the wire, not JSON
- **Progressive Enhancement**: Works without JavaScript, enhances with it
- **Zero JavaScript Required**: Backend-driven interactions
- **Composable**: Works seamlessly with Conduit templates
- **Performance**: Minimal payload sizes, efficient updates

## Phase 1: Core Protocol (Week 1-2)

### 1.1 Request/Response Protocol
```go
// Jolt request headers
const (
    HeaderJoltRequest    = "X-Jolt-Request"
    HeaderJoltTrigger    = "X-Jolt-Trigger"
    HeaderJoltTarget     = "X-Jolt-Target"
    HeaderJoltSwap       = "X-Jolt-Swap"
    HeaderJoltValues     = "X-Jolt-Values"
)

// Response types
type JoltResponse struct {
    Target     string           // CSS selector
    SwapType   SwapStrategy     // innerHTML, outerHTML, etc.
    Content    []byte           // HTML content
    Headers    map[string]string // Additional headers
    Events     []Event          // Client-side events
}

// Swap strategies
type SwapStrategy string
const (
    SwapInnerHTML    SwapStrategy = "innerHTML"
    SwapOuterHTML    SwapStrategy = "outerHTML"
    SwapBeforeBegin  SwapStrategy = "beforebegin"
    SwapAfterBegin   SwapStrategy = "afterbegin"
    SwapBeforeEnd    SwapStrategy = "beforeend"
    SwapAfterEnd     SwapStrategy = "afterend"
    SwapDelete       SwapStrategy = "delete"
)
```

### 1.2 Server-Side API
```go
package jolt

// Response builders
func Replace(target string, component conduit.Component) Response
func ReplaceOuter(target string, component conduit.Component) Response
func Append(target string, component conduit.Component) Response
func Prepend(target string, component conduit.Component) Response
func Delete(target string) Response

// Multi-target updates
func Multi(responses ...Response) Response

// Special responses
func Redirect(url string) Response
func Refresh() Response
func None() Response

// Event triggers
func Trigger(event string, detail interface{}) Response
func TriggerAfterSettle(event string) Response
func TriggerAfterSwap(event string) Response

// Headers
func PushURL(url string) Response
func ReplaceURL(url string) Response
```

### 1.3 Detection & Response
```go
// Check if request is from Jolt
func IsJoltRequest(c *bolt.Context) bool

// Send Jolt response
func Respond(c *bolt.Context, responses ...Response) error

// Stream responses
func Stream(c *bolt.Context, writer StreamWriter) error

type StreamWriter interface {
    Write(response Response) error
    Flush() error
}
```

## Phase 2: Client-Side Engine (Week 3-4)

### 2.1 Core JavaScript Library
```javascript
// jolt.js - Minimal client library (~10KB minified)

class Jolt {
    constructor(config = {}) {
        this.config = {
            defaultSwapStyle: 'innerHTML',
            defaultSwapDelay: 0,
            defaultSettleDelay: 20,
            timeout: 0,
            historyEnabled: true,
            ...config
        };
    }

    // Core methods
    ajax(verb, path, context)
    findTarget(target)
    swap(target, content, swapStyle)
    settle(target)
    trigger(event, detail)

    // Extension points
    onBeforeRequest(xhr)
    onAfterRequest(xhr)
    onBeforeSwap(target, content)
    onAfterSwap(target)
    onError(xhr, context)
}
```

### 2.2 Attribute System
```html
<!-- HTTP methods -->
<button jolt-get="/api/data">GET</button>
<form jolt-post="/api/users">POST</form>
<button jolt-put="/api/users/1">PUT</button>
<button jolt-patch="/api/users/1">PATCH</button>
<button jolt-delete="/api/users/1">DELETE</button>

<!-- Targeting -->
<div jolt-get="/content" jolt-target="#result">
<div jolt-get="/content" jolt-target="closest div">
<div jolt-get="/content" jolt-target="this">

<!-- Swapping -->
<div jolt-swap="outerHTML">Replace entire element</div>
<div jolt-swap="innerHTML">Replace content (default)</div>
<div jolt-swap="beforebegin">Insert before</div>
<div jolt-swap="afterend">Insert after</div>

<!-- Triggers -->
<input jolt-trigger="keyup changed delay:500ms">
<div jolt-trigger="revealed">Load on scroll</div>
<div jolt-trigger="intersect">Load on viewport</div>
<button jolt-trigger="click once">Fire once</button>

<!-- Indicators -->
<button jolt-indicator=".spinner">Show spinner</button>
<button jolt-indicator="closest .loading">Show loading</button>

<!-- Confirmation -->
<button jolt-confirm="Are you sure?">Confirm first</button>

<!-- Values to include -->
<div jolt-include="[name='email']">Include input values</div>
```

### 2.3 Advanced Features
```javascript
// WebSocket support
jolt.createWebSocket("/ws")

// Server-Sent Events
jolt.createEventSource("/events")

// Extension API
jolt.defineExtension("myext", {
    onEvent: function(name, evt) {},
    transformResponse: function(text, xhr, elt) {},
    isInlineSwap: function(swapStyle) {},
    handleSwap: function(swapStyle, target, fragment, settleInfo) {}
})
```

## Phase 3: Advanced Features (Week 5-6)

### 3.1 Out-of-Band Updates
```go
// Server-side
func (r *Response) AddOOB(target string, content conduit.Component)

// Response format
type OOBUpdate struct {
    Target   string
    SwapType SwapStrategy
    Content  []byte
}

// Client-side handling
// Automatically detect and apply OOB updates
```

### 3.2 Optimistic UI
```go
// Mark response as optimistic
func Optimistic(response Response) Response

// Client applies changes immediately, server confirms
type OptimisticUpdate struct {
    Target    string
    Immediate []byte  // Apply immediately
    Confirmed []byte  // Apply after server confirms
}
```

### 3.3 Request Coordination
```javascript
// Request queuing
jolt-queue="first"  // Cancel existing, run new
jolt-queue="last"   // Keep existing, queue new
jolt-queue="all"    // Run all in sequence

// Request synchronization
jolt-sync="/api/*"  // Synchronize matching requests

// Debouncing/Throttling
jolt-trigger="keyup throttle:1s"
jolt-trigger="input debounce:500ms"
```

### 3.4 History Management
```javascript
// Push/Replace URL
jolt-push-url="true"
jolt-push-url="/custom/url"
jolt-replace-url="true"

// History snapshots
jolt-history-elt  // Save element state
jolt-history="false"  // Disable history
```

## Phase 4: Performance Optimizations (Week 7)

### 4.1 Response Compression
```go
// Automatic compression detection
func CompressResponse(r Response) Response {
    if len(r.Content) > 1400 { // TCP MTU
        return gzipCompress(r)
    }
    return r
}
```

### 4.2 Caching Strategy
```go
// Server-side caching
type CacheConfig struct {
    Enabled   bool
    TTL       time.Duration
    KeyFunc   func(*bolt.Context) string
}

// ETags and 304 responses
func WithETag(response Response) Response
func CheckETag(c *bolt.Context) bool
```

### 4.3 Streaming Responses
```go
// Stream large responses
func StreamResponse(c *bolt.Context, generator func(chan Response)) error {
    ch := make(chan Response)
    go generator(ch)

    for response := range ch {
        if err := writePartial(c, response); err != nil {
            return err
        }
        c.Response.Flush()
    }
    return nil
}
```

### 4.4 Request Deduplication
```javascript
// Client-side deduplication
class RequestManager {
    pending = new Map()

    deduplicate(key, requestFn) {
        if (this.pending.has(key)) {
            return this.pending.get(key)
        }
        const promise = requestFn()
        this.pending.set(key, promise)
        promise.finally(() => this.pending.delete(key))
        return promise
    }
}
```

## Phase 5: Integration (Week 8)

### 5.1 Conduit Integration
```go
// Seamless template integration
func RenderPartial(c *bolt.Context, component conduit.Component) error {
    if IsJoltRequest(c) {
        target := c.Header(HeaderJoltTarget)
        swap := c.Header(HeaderJoltSwap)
        return Respond(c,
            Response{
                Target:   target,
                SwapType: SwapStrategy(swap),
                Content:  component.Render(),
            },
        )
    }
    return conduit.Render(c, component)
}
```

### 5.2 Bolt Middleware
```go
// Jolt middleware for automatic handling
func JoltMiddleware() bolt.Middleware {
    return func(next bolt.Handler) bolt.Handler {
        return func(c *bolt.Context) error {
            c.Set("jolt.enabled", true)
            c.Set("jolt.request", IsJoltRequest(c))
            return next(c)
        }
    }
}
```

### 5.3 Development Tools
```go
// Debug mode with verbose logging
type DebugConfig struct {
    LogRequests   bool
    LogResponses  bool
    LogSwaps      bool
    SlowWarning   time.Duration
}

// Chrome DevTools extension
// Visual debugging overlay
// Request timeline viewer
```

## Phase 6: Testing & Documentation (Week 9-10)

### 6.1 Testing Strategy
- Unit tests for all response builders
- Integration tests with Conduit
- Browser compatibility tests
- Performance benchmarks
- Load testing with concurrent requests
- E2E tests with Playwright/Cypress

### 6.2 Documentation
- Getting started guide
- API reference
- Example applications
- Migration from htmx
- Performance tuning guide
- Troubleshooting guide

### 6.3 Example Applications
```go
// 1. Todo App - CRUD operations
// 2. Chat App - Real-time updates
// 3. Dashboard - Complex interactions
// 4. Form Validation - Progressive enhancement
// 5. Infinite Scroll - Dynamic loading
// 6. Search Autocomplete - Debounced requests
```

## Performance Targets
- Client library: <10KB minified, <4KB gzipped
- Response time: <5ms server processing
- DOM updates: <16ms (60fps)
- Memory usage: <1MB for 1000 elements
- Network efficiency: 90% reduction vs full page loads

## API Examples

### Basic Usage
```go
// Handler
app.Get("/users/:id", func(c *bolt.Context) error {
    user := getUser(c.Param("id"))

    if jolt.IsJoltRequest(c) {
        return jolt.Replace("#user-card", UserCard(user))
    }

    return conduit.Render(c, UserPage(user))
})
```

### Multiple Updates
```go
app.Post("/comments", func(c *bolt.Context) error {
    comment := createComment(c)

    return jolt.Multi(
        jolt.Prepend("#comments", CommentView(comment)),
        jolt.Replace("#comment-count", CountBadge(count+1)),
        jolt.Replace("#comment-form", CommentForm{}),
        jolt.Trigger("comment-added", comment.ID),
    )
})
```

### Streaming Updates
```go
app.Get("/notifications", func(c *bolt.Context) error {
    return jolt.Stream(c, func(w jolt.StreamWriter) error {
        for notification := range notifications {
            w.Write(jolt.Prepend("#notifications",
                NotificationCard(notification)))
            w.Flush()
        }
        return nil
    })
})
```

## Success Metrics
- [ ] Core protocol implemented
- [ ] Client library <10KB minified
- [ ] All swap strategies working
- [ ] WebSocket support functional
- [ ] Zero-JavaScript mode works
- [ ] 90% network reduction achieved
- [ ] Integration tests passing
- [ ] Documentation complete
- [ ] 3+ example apps built

## Dependencies
- Conduit (templating integration)
- Bolt (framework integration)
- Electron (shared utilities)
- Compression libraries
- WebSocket libraries

## Risk Mitigation
- **Risk**: Browser compatibility issues
  - **Mitigation**: Progressive enhancement, polyfills
- **Risk**: Performance degradation with many updates
  - **Mitigation**: Request batching, virtual DOM diffing
- **Risk**: SEO concerns
  - **Mitigation**: Server-side rendering fallback
- **Risk**: Debugging complexity
  - **Mitigation**: Comprehensive dev tools, logging