package client

// ClientHeaders is an alias for CompactHeaders for API compatibility.
// The compact implementation reduces memory from 12KB to 2.2KB per instance.
type ClientHeaders = CompactHeaders
