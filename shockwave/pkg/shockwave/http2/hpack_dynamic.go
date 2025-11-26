package http2

// HPACK Dynamic Table - RFC 7541 Section 2.3
//
// The dynamic table consists of a list of header fields maintained in FIFO order.
// Entries are added to the beginning and evicted from the end when the table exceeds its size.
// Dynamic table indices start at 62 (static table is 1-61).

// dynamicTable implements the HPACK dynamic table as a circular buffer
type dynamicTable struct {
	entries  []HeaderField // Circular buffer of entries
	head     int           // Index of newest entry
	count    int           // Number of entries
	size     uint32        // Current size in bytes
	maxSize  uint32        // Maximum size in bytes
}

// entrySize calculates the size of a header field per RFC 7541 Section 4.1:
// The size of an entry is the sum of its name's length in octets,
// its value's length in octets, and 32 (overhead).
func entrySize(name, value string) uint32 {
	return uint32(len(name) + len(value) + 32)
}

// newDynamicTable creates a new dynamic table with the specified maximum size
func newDynamicTable(maxSize uint32) *dynamicTable {
	// Pre-allocate for common case (4096 bytes / ~64 bytes per entry = ~64 entries)
	capacity := int(maxSize/64)
	if capacity < 16 {
		capacity = 16
	}

	return &dynamicTable{
		entries: make([]HeaderField, capacity),
		maxSize: maxSize,
	}
}

// Add adds a new entry to the beginning of the dynamic table.
// Evicts old entries if necessary to stay within size limit.
func (dt *dynamicTable) Add(name, value string) {
	size := entrySize(name, value)

	// Evict entries if needed to make room
	for dt.size+size > dt.maxSize && dt.count > 0 {
		dt.evictOldest()
	}

	// Don't add if entry is larger than max size
	if size > dt.maxSize {
		return
	}

	// Expand buffer if needed
	if dt.count == len(dt.entries) {
		dt.resize()
	}

	// Add new entry at head
	dt.head = (dt.head - 1 + len(dt.entries)) % len(dt.entries)
	dt.entries[dt.head] = HeaderField{Name: name, Value: value}
	dt.count++
	dt.size += size
}

// Get retrieves an entry by dynamic table index (1-based, where 1 is the newest entry)
func (dt *dynamicTable) Get(index int) (HeaderField, bool) {
	if index < 1 || index > dt.count {
		return HeaderField{}, false
	}

	// Convert 1-based index to buffer position
	pos := (dt.head + index - 1) % len(dt.entries)
	return dt.entries[pos], true
}

// Find searches for a header field in the dynamic table.
// Returns (index, exactMatch) where index is 1-based (1 = newest entry).
// exactMatch is true if both name and value match, false if only name matches.
func (dt *dynamicTable) Find(name, value string) (index int, exactMatch bool) {
	for i := 0; i < dt.count; i++ {
		pos := (dt.head + i) % len(dt.entries)
		entry := dt.entries[pos]

		if entry.Name == name {
			if entry.Value == value {
				return i + 1, true // Exact match
			}
			if index == 0 {
				index = i + 1 // Name-only match (first occurrence)
			}
		}
	}

	return index, false
}

// Len returns the number of entries in the dynamic table
func (dt *dynamicTable) Len() int {
	return dt.count
}

// Size returns the current size of the dynamic table in bytes
func (dt *dynamicTable) Size() uint32 {
	return dt.size
}

// MaxSize returns the maximum size of the dynamic table in bytes
func (dt *dynamicTable) MaxSize() uint32 {
	return dt.maxSize
}

// SetMaxSize changes the maximum size of the dynamic table.
// If the new size is smaller, entries are evicted from the end.
func (dt *dynamicTable) SetMaxSize(maxSize uint32) {
	dt.maxSize = maxSize

	// Evict entries if current size exceeds new max
	for dt.size > dt.maxSize && dt.count > 0 {
		dt.evictOldest()
	}
}

// evictOldest removes the oldest entry from the dynamic table
func (dt *dynamicTable) evictOldest() {
	if dt.count == 0 {
		return
	}

	// Find oldest entry (tail)
	tail := (dt.head + dt.count - 1) % len(dt.entries)
	entry := dt.entries[tail]

	// Update size and count
	dt.size -= entrySize(entry.Name, entry.Value)
	dt.count--

	// Clear the entry (help GC)
	dt.entries[tail] = HeaderField{}
}

// resize doubles the capacity of the circular buffer
func (dt *dynamicTable) resize() {
	newSize := len(dt.entries) * 2
	newEntries := make([]HeaderField, newSize)

	// Copy existing entries to new buffer (linearize them)
	for i := 0; i < dt.count; i++ {
		pos := (dt.head + i) % len(dt.entries)
		newEntries[i] = dt.entries[pos]
	}

	dt.entries = newEntries
	dt.head = 0
}

// Reset clears all entries from the dynamic table
func (dt *dynamicTable) Reset() {
	for i := 0; i < dt.count; i++ {
		pos := (dt.head + i) % len(dt.entries)
		dt.entries[pos] = HeaderField{}
	}
	dt.head = 0
	dt.count = 0
	dt.size = 0
}

// indexTable combines static and dynamic tables for unified indexing
type indexTable struct {
	dynamic *dynamicTable
}

// newIndexTable creates a new combined index table
func newIndexTable(maxDynamicSize uint32) *indexTable {
	return &indexTable{
		dynamic: newDynamicTable(maxDynamicSize),
	}
}

// Get retrieves an entry by absolute index (1-61 = static, 62+ = dynamic)
func (it *indexTable) Get(index int) (HeaderField, bool) {
	if index <= 0 {
		return HeaderField{}, false
	}

	// Static table: indices 1-61
	if index <= StaticTableSize {
		return GetStaticEntry(index), true
	}

	// Dynamic table: indices 62+
	dynamicIndex := index - StaticTableSize
	return it.dynamic.Get(dynamicIndex)
}

// Add adds a new entry to the dynamic table
func (it *indexTable) Add(name, value string) {
	it.dynamic.Add(name, value)
}

// Find searches both static and dynamic tables for a header field.
// Returns (index, exactMatch) where index is absolute (1-61 = static, 62+ = dynamic).
func (it *indexTable) Find(name, value string) (index int, exactMatch bool) {
	// Search static table first
	staticIdx, staticExact := FindStaticIndex(name, value)
	if staticExact {
		return staticIdx, true
	}

	// Search dynamic table
	dynamicIdx, dynamicExact := it.dynamic.Find(name, value)
	if dynamicIdx > 0 {
		// Convert to absolute index
		absoluteIdx := StaticTableSize + dynamicIdx
		if dynamicExact {
			return absoluteIdx, true
		}
		// Have dynamic name match
		if staticIdx == 0 {
			// No static match, use dynamic
			return absoluteIdx, false
		}
	}

	// Return static name match if found
	if staticIdx > 0 {
		return staticIdx, false
	}

	return 0, false
}

// SetMaxDynamicSize changes the maximum size of the dynamic table
func (it *indexTable) SetMaxDynamicSize(maxSize uint32) {
	it.dynamic.SetMaxSize(maxSize)
}

// DynamicTableSize returns the current size of the dynamic table
func (it *indexTable) DynamicTableSize() uint32 {
	return it.dynamic.Size()
}
