package http2

// HPACK Static Table - RFC 7541 Appendix A
//
// The static table consists of 61 predefined header field entries.
// These entries are never evicted and are indexed starting from 1.

// HeaderField represents a header name-value pair
type HeaderField struct {
	Name  string
	Value string
}

// staticTable is the HPACK static table defined in RFC 7541 Appendix A.
// Index 0 is unused; valid indices are 1-61.
var staticTable = [...]HeaderField{
	{},                                           // Index 0 - unused
	{":authority", ""},                           // 1
	{":method", "GET"},                           // 2
	{":method", "POST"},                          // 3
	{":path", "/"},                               // 4
	{":path", "/index.html"},                     // 5
	{":scheme", "http"},                          // 6
	{":scheme", "https"},                         // 7
	{":status", "200"},                           // 8
	{":status", "204"},                           // 9
	{":status", "206"},                           // 10
	{":status", "304"},                           // 11
	{":status", "400"},                           // 12
	{":status", "404"},                           // 13
	{":status", "500"},                           // 14
	{"accept-charset", ""},                       // 15
	{"accept-encoding", "gzip, deflate"},         // 16
	{"accept-language", ""},                      // 17
	{"accept-ranges", ""},                        // 18
	{"accept", ""},                               // 19
	{"access-control-allow-origin", ""},          // 20
	{"age", ""},                                  // 21
	{"allow", ""},                                // 22
	{"authorization", ""},                        // 23
	{"cache-control", ""},                        // 24
	{"content-disposition", ""},                  // 25
	{"content-encoding", ""},                     // 26
	{"content-language", ""},                     // 27
	{"content-length", ""},                       // 28
	{"content-location", ""},                     // 29
	{"content-range", ""},                        // 30
	{"content-type", ""},                         // 31
	{"cookie", ""},                               // 32
	{"date", ""},                                 // 33
	{"etag", ""},                                 // 34
	{"expect", ""},                               // 35
	{"expires", ""},                              // 36
	{"from", ""},                                 // 37
	{"host", ""},                                 // 38
	{"if-match", ""},                             // 39
	{"if-modified-since", ""},                    // 40
	{"if-none-match", ""},                        // 41
	{"if-range", ""},                             // 42
	{"if-unmodified-since", ""},                  // 43
	{"last-modified", ""},                        // 44
	{"link", ""},                                 // 45
	{"location", ""},                             // 46
	{"max-forwards", ""},                         // 47
	{"proxy-authenticate", ""},                   // 48
	{"proxy-authorization", ""},                  // 49
	{"range", ""},                                // 50
	{"referer", ""},                              // 51
	{"refresh", ""},                              // 52
	{"retry-after", ""},                          // 53
	{"server", ""},                               // 54
	{"set-cookie", ""},                           // 55
	{"strict-transport-security", ""},            // 56
	{"transfer-encoding", ""},                    // 57
	{"user-agent", ""},                           // 58
	{"vary", ""},                                 // 59
	{"via", ""},                                  // 60
	{"www-authenticate", ""},                     // 61
}

// StaticTableSize is the number of entries in the static table
const StaticTableSize = 61

// GetStaticEntry returns the static table entry at the given index (1-61).
// Returns an empty HeaderField if index is out of range.
func GetStaticEntry(index int) HeaderField {
	if index < 1 || index > StaticTableSize {
		return HeaderField{}
	}
	return staticTable[index]
}

// staticTableLookup is a pre-computed map for fast static table lookups
// Maps name or name:value to index
var staticTableLookup map[string]int

func init() {
	// Build lookup map for O(1) static table searches
	staticTableLookup = make(map[string]int, StaticTableSize*2)

	for i := 1; i <= StaticTableSize; i++ {
		entry := staticTable[i]

		// Index by name only (for name matches)
		key := entry.Name
		if _, exists := staticTableLookup[key]; !exists {
			// Store first occurrence of this name
			staticTableLookup[key] = i
		}

		// Index by name:value (for exact matches)
		if entry.Value != "" {
			fullKey := entry.Name + "\x00" + entry.Value
			staticTableLookup[fullKey] = i
		}
	}
}

// FindStaticIndex searches the static table for a header field.
// Returns (index, exactMatch) where:
// - index is the static table index (1-61), or 0 if not found
// - exactMatch is true if both name and value match, false if only name matches
func FindStaticIndex(name, value string) (index int, exactMatch bool) {
	// Try exact match first (name + value)
	if value != "" {
		fullKey := name + "\x00" + value
		if idx, found := staticTableLookup[fullKey]; found {
			return idx, true
		}
	}

	// Try name-only match
	if idx, found := staticTableLookup[name]; found {
		return idx, false
	}

	return 0, false
}
