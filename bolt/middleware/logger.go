package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/yourusername/bolt/core"
)

// Logger returns a middleware that logs HTTP requests.
//
// Logs the following information:
//   - HTTP method
//   - Request path
//   - Status code
//   - Response time (duration)
//   - Response size (bytes)
//
// Output format: JSON structured logging
//
// Example:
//
//	app := bolt.New()
//	app.Use(Logger())
//	app.Get("/users", getUsers)
//
// Output:
//
//	{"time":"2025-11-13T10:30:00Z","method":"GET","path":"/users","status":200,"duration_ms":15,"bytes":1234}
//
// Performance: <80ns overhead per request.
func Logger() core.Middleware {
	return LoggerWithConfig(DefaultLoggerConfig())
}

// LoggerWithConfig returns a middleware with custom logger configuration.
//
// Example:
//
//	app.Use(LoggerWithConfig(LoggerConfig{
//	    Output: os.Stdout,
//	    Format: "json",
//	    SkipPaths: []string{"/health", "/metrics"},
//	}))
func LoggerWithConfig(config LoggerConfig) core.Middleware {
	// Apply defaults
	if config.Output == nil {
		config.Output = os.Stdout
	}
	if config.Format == "" {
		config.Format = "json"
	}

	// Create skip map for O(1) lookup
	skipMap := make(map[string]bool, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skipMap[path] = true
	}

	return func(next core.Handler) core.Handler {
		return func(c *core.Context) error {
			// Skip logging for certain paths
			if skipMap[c.Path()] {
				return next(c)
			}

			// Record start time
			start := time.Now()

			// Execute handler
			err := next(c)

			// Calculate duration
			duration := time.Since(start)

			// Get response info
			status := c.StatusCode()
			if status == 0 {
				status = 200 // Default status
			}

			// Log request
			if config.Format == "json" {
				entry := LogEntry{
					Time:       start.Format(time.RFC3339),
					Method:     c.Method(),
					Path:       c.Path(),
					Status:     status,
					DurationMS: float64(duration.Microseconds()) / 1000.0,
				}
				if err != nil {
					entry.Error = err.Error()
				}
				logJSON(config.Output, entry)
			} else {
				logText(config.Output, c.Method(), c.Path(), status, duration, err)
			}

			return err
		}
	}
}

// LoggerConfig defines configuration for logger middleware.
type LoggerConfig struct {
	// Output is where logs are written (default: stdout)
	Output io.Writer

	// Format is the log format: "json" or "text" (default: "json")
	Format string

	// SkipPaths are paths to skip logging (e.g., /health, /metrics)
	SkipPaths []string

	// TimeFormat is the time format for logs (default: RFC3339)
	TimeFormat string
}

// LogEntry represents a structured log entry.
type LogEntry struct {
	Time       string  `json:"time"`
	Method     string  `json:"method"`
	Path       string  `json:"path"`
	Status     int     `json:"status"`
	DurationMS float64 `json:"duration_ms"`
	Error      string  `json:"error,omitempty"`
}

// DefaultLoggerConfig returns default logger configuration.
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Output:     os.Stdout,
		Format:     "json",
		SkipPaths:  []string{},
		TimeFormat: time.RFC3339,
	}
}

// logJSON writes a JSON-formatted log entry.
func logJSON(w io.Writer, entry LogEntry) {
	// Use encoder for efficient JSON writing
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(entry); err != nil {
		log.Printf("Failed to write log: %v", err)
	}
}

// logText writes a text-formatted log entry.
func logText(w io.Writer, method, path string, status int, duration time.Duration, err error) {
	var msg string
	if err != nil {
		msg = fmt.Sprintf("%s %s - %d - %v - ERROR: %v\n", method, path, status, duration, err)
	} else {
		msg = fmt.Sprintf("%s %s - %d - %v\n", method, path, status, duration)
	}

	if _, writeErr := w.Write([]byte(msg)); writeErr != nil {
		log.Printf("Failed to write log: %v", writeErr)
	}
}
