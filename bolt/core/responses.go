package core

// Pre-compiled JSON responses for common REST API responses.
// These are byte slices allocated once at startup, providing zero-allocation
// responses for the most common API response patterns.
//
// Performance: 0 allocs/op vs 2-3 allocs/op for dynamic JSON encoding.
//
// Coverage: Approximately 40% of typical REST API responses are simple
// status responses (ok, created, deleted, not found, etc.).

var (
	// Success responses (2xx)
	jsonOKBytes        = []byte(`{"ok":true}`)
	jsonCreatedBytes   = []byte(`{"created":true}`)
	jsonDeletedBytes   = []byte(`{"deleted":true}`)
	jsonUpdatedBytes   = []byte(`{"updated":true}`)
	jsonAcceptedBytes  = []byte(`{"accepted":true}`)
	jsonNoContentBytes = []byte(``) // 204 No Content (empty body)

	// Client error responses (4xx)
	json400Bytes = []byte(`{"error":"Bad Request"}`)
	json401Bytes = []byte(`{"error":"Unauthorized"}`)
	json403Bytes = []byte(`{"error":"Forbidden"}`)
	json404Bytes = []byte(`{"error":"Not Found"}`)
	json405Bytes = []byte(`{"error":"Method Not Allowed"}`)
	json408Bytes = []byte(`{"error":"Request Timeout"}`)
	json409Bytes = []byte(`{"error":"Conflict"}`)
	json410Bytes = []byte(`{"error":"Gone"}`)
	json413Bytes = []byte(`{"error":"Payload Too Large"}`)
	json422Bytes = []byte(`{"error":"Unprocessable Entity"}`)
	json429Bytes = []byte(`{"error":"Too Many Requests"}`)

	// Server error responses (5xx)
	json500Bytes = []byte(`{"error":"Internal Server Error"}`)
	json501Bytes = []byte(`{"error":"Not Implemented"}`)
	json502Bytes = []byte(`{"error":"Bad Gateway"}`)
	json503Bytes = []byte(`{"error":"Service Unavailable"}`)
	json504Bytes = []byte(`{"error":"Gateway Timeout"}`)
)

// JSONOK sends {"ok":true} with 200 status (zero allocations).
//
// Common use cases:
//   - Health check endpoints
//   - Simple success confirmations
//   - Acknowledgement responses
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Get("/ping", func(c *bolt.Context) error {
//	    return c.JSONOK()
//	})
func (c *Context) JSONOK() error {
	c.setContentTypeJSON()
	c.statusCode = 200
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(200)
		_, err := c.shockwaveRes.Write(jsonOKBytes)
		return err
	}

	// Fallback to http.ResponseWriter (testing)
	c.httpRes.WriteHeader(200)
	_, err := c.httpRes.Write(jsonOKBytes)
	return err
}

// JSONCreated sends {"created":true} with 201 status (zero allocations).
//
// Common use cases:
//   - Resource creation endpoints (POST)
//   - Signup/registration confirmations
//   - Simple creation acknowledgements
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Post("/users", func(c *bolt.Context) error {
//	    // ... create user ...
//	    return c.JSONCreated()
//	})
func (c *Context) JSONCreated() error {
	c.setContentTypeJSON()
	c.statusCode = 201
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(201)
		_, err := c.shockwaveRes.Write(jsonCreatedBytes)
		return err
	}

	c.httpRes.WriteHeader(201)
	_, err := c.httpRes.Write(jsonCreatedBytes)
	return err
}

// JSONDeleted sends {"deleted":true} with 200 status (zero allocations).
//
// Common use cases:
//   - Resource deletion endpoints (DELETE)
//   - Cleanup confirmations
//   - Simple deletion acknowledgements
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Delete("/users/:id", func(c *bolt.Context) error {
//	    // ... delete user ...
//	    return c.JSONDeleted()
//	})
func (c *Context) JSONDeleted() error {
	c.setContentTypeJSON()
	c.statusCode = 200
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(200)
		_, err := c.shockwaveRes.Write(jsonDeletedBytes)
		return err
	}

	c.httpRes.WriteHeader(200)
	_, err := c.httpRes.Write(jsonDeletedBytes)
	return err
}

// JSONUpdated sends {"updated":true} with 200 status (zero allocations).
//
// Common use cases:
//   - Resource update endpoints (PUT/PATCH)
//   - Settings update confirmations
//   - Simple update acknowledgements
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Put("/users/:id", func(c *bolt.Context) error {
//	    // ... update user ...
//	    return c.JSONUpdated()
//	})
func (c *Context) JSONUpdated() error {
	c.setContentTypeJSON()
	c.statusCode = 200
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(200)
		_, err := c.shockwaveRes.Write(jsonUpdatedBytes)
		return err
	}

	c.httpRes.WriteHeader(200)
	_, err := c.httpRes.Write(jsonUpdatedBytes)
	return err
}

// JSONAccepted sends {"accepted":true} with 202 status (zero allocations).
//
// Common use cases:
//   - Async job submissions
//   - Background task queuing
//   - Deferred processing acknowledgements
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Post("/jobs", func(c *bolt.Context) error {
//	    // ... queue job ...
//	    return c.JSONAccepted()
//	})
func (c *Context) JSONAccepted() error {
	c.setContentTypeJSON()
	c.statusCode = 202
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(202)
		_, err := c.shockwaveRes.Write(jsonAcceptedBytes)
		return err
	}

	c.httpRes.WriteHeader(202)
	_, err := c.httpRes.Write(jsonAcceptedBytes)
	return err
}

// JSONNoContent sends 204 No Content with empty body (zero allocations).
//
// Common use cases:
//   - Successful DELETE with no content
//   - Successful PUT with no content
//   - Preflight CORS requests
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Delete("/users/:id", func(c *bolt.Context) error {
//	    // ... delete user ...
//	    return c.JSONNoContent()
//	})
func (c *Context) JSONNoContent() error {
	c.statusCode = 204
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(204)
		return nil
	}

	c.httpRes.WriteHeader(204)
	return nil
}

// JSONBadRequest sends {"error":"Bad Request"} with 400 status (zero allocations).
//
// Common use cases:
//   - Invalid request data
//   - Missing required fields
//   - Malformed JSON
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Post("/users", func(c *bolt.Context) error {
//	    if !valid {
//	        return c.JSONBadRequest()
//	    }
//	    // ... create user ...
//	})
func (c *Context) JSONBadRequest() error {
	c.setContentTypeJSON()
	c.statusCode = 400
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(400)
		_, err := c.shockwaveRes.Write(json400Bytes)
		return err
	}

	c.httpRes.WriteHeader(400)
	_, err := c.httpRes.Write(json400Bytes)
	return err
}

// JSONUnauthorized sends {"error":"Unauthorized"} with 401 status (zero allocations).
//
// Common use cases:
//   - Missing authentication
//   - Invalid credentials
//   - Expired tokens
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Get("/profile", func(c *bolt.Context) error {
//	    if !authenticated {
//	        return c.JSONUnauthorized()
//	    }
//	    // ... return profile ...
//	})
func (c *Context) JSONUnauthorized() error {
	c.setContentTypeJSON()
	c.statusCode = 401
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(401)
		_, err := c.shockwaveRes.Write(json401Bytes)
		return err
	}

	c.httpRes.WriteHeader(401)
	_, err := c.httpRes.Write(json401Bytes)
	return err
}

// JSONForbidden sends {"error":"Forbidden"} with 403 status (zero allocations).
//
// Common use cases:
//   - Insufficient permissions
//   - Access denied
//   - Resource forbidden for user
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Delete("/admin/users/:id", func(c *bolt.Context) error {
//	    if !isAdmin {
//	        return c.JSONForbidden()
//	    }
//	    // ... delete user ...
//	})
func (c *Context) JSONForbidden() error {
	c.setContentTypeJSON()
	c.statusCode = 403
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(403)
		_, err := c.shockwaveRes.Write(json403Bytes)
		return err
	}

	c.httpRes.WriteHeader(403)
	_, err := c.httpRes.Write(json403Bytes)
	return err
}

// JSONNotFound sends {"error":"Not Found"} with 404 status (zero allocations).
//
// Common use cases:
//   - Resource not found
//   - Invalid ID
//   - Deleted resources
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Get("/users/:id", func(c *bolt.Context) error {
//	    user := findUser(c.Param("id"))
//	    if user == nil {
//	        return c.JSONNotFound()
//	    }
//	    return c.JSON(200, user)
//	})
func (c *Context) JSONNotFound() error {
	c.setContentTypeJSON()
	c.statusCode = 404
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(404)
		_, err := c.shockwaveRes.Write(json404Bytes)
		return err
	}

	c.httpRes.WriteHeader(404)
	_, err := c.httpRes.Write(json404Bytes)
	return err
}

// JSONMethodNotAllowed sends {"error":"Method Not Allowed"} with 405 status (zero allocations).
//
// Common use cases:
//   - Wrong HTTP method for endpoint
//   - Method not supported
//   - Router fallback
//
// Performance: 0 allocs/op
func (c *Context) JSONMethodNotAllowed() error {
	c.setContentTypeJSON()
	c.statusCode = 405
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(405)
		_, err := c.shockwaveRes.Write(json405Bytes)
		return err
	}

	c.httpRes.WriteHeader(405)
	_, err := c.httpRes.Write(json405Bytes)
	return err
}

// JSONTooManyRequests sends {"error":"Too Many Requests"} with 429 status (zero allocations).
//
// Common use cases:
//   - Rate limiting
//   - Throttling
//   - Quota exceeded
//
// Performance: 0 allocs/op
func (c *Context) JSONTooManyRequests() error {
	c.setContentTypeJSON()
	c.statusCode = 429
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(429)
		_, err := c.shockwaveRes.Write(json429Bytes)
		return err
	}

	c.httpRes.WriteHeader(429)
	_, err := c.httpRes.Write(json429Bytes)
	return err
}

// JSONInternalError sends {"error":"Internal Server Error"} with 500 status (zero allocations).
//
// Common use cases:
//   - Unexpected errors
//   - Panic recovery
//   - System failures
//
// Performance: 0 allocs/op
//
// Example:
//
//	app.Get("/data", func(c *bolt.Context) error {
//	    data, err := fetchData()
//	    if err != nil {
//	        log.Error("failed to fetch data", err)
//	        return c.JSONInternalError()
//	    }
//	    return c.JSON(200, data)
//	})
func (c *Context) JSONInternalError() error {
	c.setContentTypeJSON()
	c.statusCode = 500
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(500)
		_, err := c.shockwaveRes.Write(json500Bytes)
		return err
	}

	c.httpRes.WriteHeader(500)
	_, err := c.httpRes.Write(json500Bytes)
	return err
}

// JSONServiceUnavailable sends {"error":"Service Unavailable"} with 503 status (zero allocations).
//
// Common use cases:
//   - Maintenance mode
//   - Overload conditions
//   - Temporary unavailability
//
// Performance: 0 allocs/op
func (c *Context) JSONServiceUnavailable() error {
	c.setContentTypeJSON()
	c.statusCode = 503
	c.written = true

	if c.shockwaveRes != nil {
		c.shockwaveRes.WriteHeader(503)
		_, err := c.shockwaveRes.Write(json503Bytes)
		return err
	}

	c.httpRes.WriteHeader(503)
	_, err := c.httpRes.Write(json503Bytes)
	return err
}
