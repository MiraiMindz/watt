package qpack

import (
	"errors"
	"sync"
)

// RFC 9204 Section 3: QPACK Dynamic Table

var (
	ErrTableFull    = errors.New("qpack: dynamic table full")
	ErrInvalidIndex = errors.New("qpack: invalid index")
)

// DynamicTableEntry represents an entry in the dynamic table
type DynamicTableEntry struct {
	Name  string
	Value string
	Size  uint64 // Size in bytes (name length + value length + 32)
}

// CalculateEntrySize calculates the size of an entry
// RFC 9204 Section 3.2.1: size = len(name) + len(value) + 32
func CalculateEntrySize(name, value string) uint64 {
	return uint64(len(name) + len(value) + 32)
}

// DynamicTable represents the QPACK dynamic table
type DynamicTable struct {
	mu sync.RWMutex

	// Table configuration
	maxSize     uint64 // Maximum table size in bytes
	currentSize uint64 // Current table size in bytes

	// Entries (circular buffer)
	entries     []DynamicTableEntry
	insertIndex uint64 // Absolute insert index
	baseIndex   uint64 // Base index (oldest entry)

	// Capacity
	capacity int
}

// NewDynamicTable creates a new dynamic table with the given maximum size
func NewDynamicTable(maxSize uint64) *DynamicTable {
	return &DynamicTable{
		maxSize:     maxSize,
		currentSize: 0,
		entries:     make([]DynamicTableEntry, 0, 64),
		insertIndex: 0,
		baseIndex:   0,
		capacity:    64,
	}
}

// SetMaxSize updates the maximum table size
func (dt *DynamicTable) SetMaxSize(maxSize uint64) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.maxSize = maxSize

	// Evict entries if necessary
	dt.evictToSize(maxSize)
}

// GetMaxSize returns the maximum table size
func (dt *DynamicTable) GetMaxSize() uint64 {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.maxSize
}

// GetCurrentSize returns the current table size
func (dt *DynamicTable) GetCurrentSize() uint64 {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.currentSize
}

// Insert adds a new entry to the dynamic table
func (dt *DynamicTable) Insert(name, value string) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	entrySize := CalculateEntrySize(name, value)

	// Check if entry is too large for table
	if entrySize > dt.maxSize {
		return ErrTableFull
	}

	// Evict entries to make room
	if err := dt.evictToSize(dt.maxSize - entrySize); err != nil {
		return err
	}

	// Add entry
	entry := DynamicTableEntry{
		Name:  name,
		Value: value,
		Size:  entrySize,
	}

	dt.entries = append(dt.entries, entry)
	dt.currentSize += entrySize
	dt.insertIndex++

	return nil
}

// Get retrieves an entry by absolute index
func (dt *DynamicTable) Get(absoluteIndex uint64) (DynamicTableEntry, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	// Check if index is valid
	if absoluteIndex < dt.baseIndex || absoluteIndex >= dt.insertIndex {
		return DynamicTableEntry{}, ErrInvalidIndex
	}

	// Convert absolute index to relative index
	relativeIndex := int(absoluteIndex - dt.baseIndex)
	if relativeIndex >= len(dt.entries) {
		return DynamicTableEntry{}, ErrInvalidIndex
	}

	return dt.entries[relativeIndex], nil
}

// Find searches for an entry matching name and optionally value
// Returns (absoluteIndex, exactMatch)
func (dt *DynamicTable) Find(name, value string) (uint64, bool) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var nameOnlyMatch uint64 = 0
	nameOnlyFound := false

	for i, entry := range dt.entries {
		if entry.Name == name {
			absoluteIndex := dt.baseIndex + uint64(i)
			if entry.Value == value {
				// Exact match
				return absoluteIndex, true
			}
			if !nameOnlyFound {
				nameOnlyMatch = absoluteIndex
				nameOnlyFound = true
			}
		}
	}

	if nameOnlyFound {
		return nameOnlyMatch, false
	}

	return 0, false
}

// Evict removes the oldest entry from the table
func (dt *DynamicTable) Evict() error {
	if len(dt.entries) == 0 {
		return nil
	}

	// Remove oldest entry (first in slice)
	entry := dt.entries[0]
	dt.entries = dt.entries[1:]
	dt.currentSize -= entry.Size
	dt.baseIndex++

	return nil
}

// evictToSize evicts entries until the table size is at most targetSize
func (dt *DynamicTable) evictToSize(targetSize uint64) error {
	for dt.currentSize > targetSize && len(dt.entries) > 0 {
		if err := dt.Evict(); err != nil {
			return err
		}
	}
	return nil
}

// Clear removes all entries from the table
func (dt *DynamicTable) Clear() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.entries = dt.entries[:0]
	dt.currentSize = 0
	dt.baseIndex = dt.insertIndex
}

// Length returns the number of entries in the table
func (dt *DynamicTable) Length() int {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return len(dt.entries)
}

// GetInsertIndex returns the current insert index
func (dt *DynamicTable) GetInsertIndex() uint64 {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.insertIndex
}

// GetBaseIndex returns the base index (oldest entry)
func (dt *DynamicTable) GetBaseIndex() uint64 {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.baseIndex
}

// Duplicate duplicates an existing entry (for QPACK duplicate instruction)
func (dt *DynamicTable) Duplicate(absoluteIndex uint64) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// Get the entry to duplicate
	if absoluteIndex < dt.baseIndex || absoluteIndex >= dt.insertIndex {
		return ErrInvalidIndex
	}

	relativeIndex := int(absoluteIndex - dt.baseIndex)
	if relativeIndex >= len(dt.entries) {
		return ErrInvalidIndex
	}

	entry := dt.entries[relativeIndex]

	// Check if we have room
	if entry.Size > dt.maxSize-dt.currentSize {
		// Evict entries to make room
		if err := dt.evictToSize(dt.maxSize - entry.Size); err != nil {
			return err
		}
	}

	// Add duplicate
	newEntry := DynamicTableEntry{
		Name:  entry.Name,
		Value: entry.Value,
		Size:  entry.Size,
	}

	dt.entries = append(dt.entries, newEntry)
	dt.currentSize += newEntry.Size
	dt.insertIndex++

	return nil
}

// GetAllEntries returns all entries in the table (for debugging)
func (dt *DynamicTable) GetAllEntries() []DynamicTableEntry {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	entries := make([]DynamicTableEntry, len(dt.entries))
	copy(entries, dt.entries)
	return entries
}
