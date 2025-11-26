package qpack

// QPACK Static Table (RFC 9204 Appendix A)
// This table is used for header compression in HTTP/3

// StaticTableEntry represents an entry in the QPACK static table
type StaticTableEntry struct {
	Name  string
	Value string
}

// staticTable is the QPACK static table as defined in RFC 9204 Appendix A
var staticTable = []StaticTableEntry{
	// Index 0-14: :authority, :path, age, etc.
	{":authority", ""},                               // 0
	{":path", "/"},                                   // 1
	{"age", "0"},                                     // 2
	{"content-disposition", ""},                      // 3
	{"content-length", "0"},                          // 4
	{"cookie", ""},                                   // 5
	{"date", ""},                                     // 6
	{"etag", ""},                                     // 7
	{"if-modified-since", ""},                        // 8
	{"if-none-match", ""},                            // 9
	{"last-modified", ""},                            // 10
	{"link", ""},                                     // 11
	{"location", ""},                                 // 12
	{"referer", ""},                                  // 13
	{"set-cookie", ""},                               // 14

	// Index 15-24: :method values
	{":method", "CONNECT"},                           // 15
	{":method", "DELETE"},                            // 16
	{":method", "GET"},                               // 17
	{":method", "HEAD"},                              // 18
	{":method", "OPTIONS"},                           // 19
	{":method", "POST"},                              // 20
	{":method", "PUT"},                               // 21
	{":scheme", "http"},                              // 22
	{":scheme", "https"},                             // 23
	{":status", "103"},                               // 24

	// Index 25-62: :status codes and common headers
	{":status", "200"},                               // 25
	{":status", "304"},                               // 26
	{":status", "404"},                               // 27
	{":status", "503"},                               // 28
	{"accept", "*/*"},                                // 29
	{"accept", "application/dns-message"},            // 30
	{"accept-encoding", "gzip, deflate, br"},         // 31
	{"accept-ranges", "bytes"},                       // 32
	{"access-control-allow-headers", "cache-control"}, // 33
	{"access-control-allow-headers", "content-type"}, // 34
	{"access-control-allow-origin", "*"},             // 35
	{"cache-control", "max-age=0"},                   // 36
	{"cache-control", "max-age=2592000"},             // 37
	{"cache-control", "max-age=604800"},              // 38
	{"cache-control", "no-cache"},                    // 39
	{"cache-control", "no-store"},                    // 40
	{"cache-control", "public, max-age=31536000"},    // 41
	{"content-encoding", "br"},                       // 42
	{"content-encoding", "gzip"},                     // 43
	{"content-type", "application/dns-message"},      // 44
	{"content-type", "application/javascript"},       // 45
	{"content-type", "application/json"},             // 46
	{"content-type", "application/x-www-form-urlencoded"}, // 47
	{"content-type", "image/gif"},                    // 48
	{"content-type", "image/jpeg"},                   // 49
	{"content-type", "image/png"},                    // 50
	{"content-type", "text/css"},                     // 51
	{"content-type", "text/html; charset=utf-8"},     // 52
	{"content-type", "text/plain"},                   // 53
	{"content-type", "text/plain;charset=utf-8"},     // 54
	{"range", "bytes=0-"},                            // 55
	{"strict-transport-security", "max-age=31536000"}, // 56

	// Index 57-98: Additional common headers
	{"strict-transport-security", "max-age=31536000; includesubdomains"}, // 57
	{"strict-transport-security", "max-age=31536000; includesubdomains; preload"}, // 58
	{"vary", "accept-encoding"},                      // 59
	{"vary", "origin"},                               // 60
	{"x-content-type-options", "nosniff"},            // 61
	{"x-xss-protection", "1; mode=block"},            // 62
	{":status", "100"},                               // 63
	{":status", "204"},                               // 64
	{":status", "206"},                               // 65
	{":status", "302"},                               // 66
	{":status", "400"},                               // 67
	{":status", "403"},                               // 68
	{":status", "421"},                               // 69
	{":status", "425"},                               // 70
	{":status", "500"},                               // 71
	{"accept-language", ""},                          // 72
	{"access-control-allow-credentials", "FALSE"},    // 73
	{"access-control-allow-credentials", "TRUE"},     // 74
	{"access-control-allow-headers", "*"},            // 75
	{"access-control-allow-methods", "get"},          // 76
	{"access-control-allow-methods", "get, post, options"}, // 77
	{"access-control-allow-methods", "options"},      // 78
	{"access-control-expose-headers", "content-length"}, // 79
	{"access-control-request-headers", "content-type"}, // 80
	{"access-control-request-method", "get"},         // 81
	{"access-control-request-method", "post"},        // 82
	{"alt-svc", "clear"},                             // 83
	{"authorization", ""},                            // 84
	{"content-security-policy", "script-src 'none'; object-src 'none'; base-uri 'none'"}, // 85
	{"early-data", "1"},                              // 86
	{"expect-ct", ""},                                // 87
	{"forwarded", ""},                                // 88
	{"if-range", ""},                                 // 89
	{"origin", ""},                                   // 90
	{"purpose", "prefetch"},                          // 91
	{"server", ""},                                   // 92
	{"timing-allow-origin", "*"},                     // 93
	{"upgrade-insecure-requests", "1"},               // 94
	{"user-agent", ""},                               // 95
	{"x-forwarded-for", ""},                          // 96
	{"x-frame-options", "deny"},                      // 97
	{"x-frame-options", "sameorigin"},                // 98
}

// StaticTableSize returns the number of entries in the static table
func StaticTableSize() int {
	return len(staticTable)
}

// GetStaticEntry returns the entry at the given index in the static table
func GetStaticEntry(index int) (StaticTableEntry, bool) {
	if index < 0 || index >= len(staticTable) {
		return StaticTableEntry{}, false
	}
	return staticTable[index], true
}

// FindStaticIndex finds the index of an exact name:value match in the static table
// Returns (index, exactMatch=true) if both name and value match
// Returns (index, exactMatch=false) if only name matches
// Returns (-1, false) if no match found
func FindStaticIndex(name, value string) (int, bool) {
	nameOnlyMatch := -1

	for i, entry := range staticTable {
		if entry.Name == name {
			if entry.Value == value {
				// Exact match
				return i, true
			}
			if nameOnlyMatch == -1 {
				// First name-only match
				nameOnlyMatch = i
			}
		}
	}

	if nameOnlyMatch != -1 {
		return nameOnlyMatch, false
	}

	return -1, false
}

// LookupByName returns all static table indices that match the given name
func LookupByName(name string) []int {
	var indices []int
	for i, entry := range staticTable {
		if entry.Name == name {
			indices = append(indices, i)
		}
	}
	return indices
}

// StaticTableIterator allows iteration over the static table
type StaticTableIterator struct {
	index int
}

// NewStaticTableIterator creates a new iterator for the static table
func NewStaticTableIterator() *StaticTableIterator {
	return &StaticTableIterator{index: 0}
}

// Next returns the next entry in the static table
func (it *StaticTableIterator) Next() (StaticTableEntry, bool) {
	if it.index >= len(staticTable) {
		return StaticTableEntry{}, false
	}
	entry := staticTable[it.index]
	it.index++
	return entry, true
}

// Reset resets the iterator to the beginning
func (it *StaticTableIterator) Reset() {
	it.index = 0
}
