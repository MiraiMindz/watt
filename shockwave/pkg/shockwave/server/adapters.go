package server

import (
	"github.com/yourusername/shockwave/pkg/shockwave/http11"
)

// responseWriterAdapter adapts http11.ResponseWriter to server.ResponseWriter interface
type responseWriterAdapter struct {
	rw *http11.ResponseWriter
}

func (w *responseWriterAdapter) Header() Header {
	// Return header adapter (allocates only if Header() is called)
	h := headerAdapterPool.Get().(*headerAdapter)
	h.h = w.rw.Header()
	return h
}

func (w *responseWriterAdapter) WriteHeader(statusCode int) {
	w.rw.WriteHeader(statusCode)
}

func (w *responseWriterAdapter) Write(data []byte) (int, error) {
	return w.rw.Write(data)
}

func (w *responseWriterAdapter) WriteString(s string) (int, error) {
	// Try to use WriteString if available (zero-copy)
	if ws, ok := interface{}(w.rw).(interface{ WriteString(string) (int, error) }); ok {
		return ws.WriteString(s)
	}
	// Fallback: this will allocate, but only if WriteString isn't available
	return w.rw.Write([]byte(s))
}

func (w *responseWriterAdapter) WriteJSON(statusCode int, data []byte) error {
	return w.rw.WriteJSON(statusCode, data)
}

func (w *responseWriterAdapter) Flush() error {
	return w.rw.Flush()
}

// headerAdapter adapts http11.Header to server.Header interface
type headerAdapter struct {
	h *http11.Header
}

func (h *headerAdapter) Get(key string) string {
	return h.h.GetString([]byte(key))
}

func (h *headerAdapter) Set(key, value string) {
	h.h.Set([]byte(key), []byte(value))
}

func (h *headerAdapter) Add(key, value string) {
	h.h.Add([]byte(key), []byte(value))
}

func (h *headerAdapter) Del(key string) {
	h.h.Del([]byte(key))
}

func (h *headerAdapter) Clone() Header {
	cloned := &http11.Header{}
	h.h.VisitAll(func(name, value []byte) bool {
		cloned.Set(name, value)
		return true
	})
	return &headerAdapter{h: cloned}
}
