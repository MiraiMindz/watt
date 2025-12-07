package memory

// lruNode is a node in the LRU doubly-linked list.
type lruNode[K comparable] struct {
	key  K
	prev *lruNode[K]
	next *lruNode[K]
}

// lruList is a doubly-linked list for LRU tracking.
// Most recently used items are at the front.
type lruList[K comparable] struct {
	head *lruNode[K]
	tail *lruNode[K]
	size int
}

// newLRUList creates a new LRU list.
func newLRUList[K comparable]() *lruList[K] {
	return &lruList[K]{}
}

// pushFront adds a node to the front of the list (most recently used).
func (l *lruList[K]) pushFront(key K) *lruNode[K] {
	node := &lruNode[K]{key: key}

	if l.head == nil {
		// Empty list
		l.head = node
		l.tail = node
	} else {
		// Add to front
		node.next = l.head
		l.head.prev = node
		l.head = node
	}

	l.size++
	return node
}

// remove removes a node from the list.
func (l *lruList[K]) remove(node *lruNode[K]) {
	if node == nil {
		return
	}

	if node.prev != nil {
		node.prev.next = node.next
	} else {
		// Removing head
		l.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		// Removing tail
		l.tail = node.prev
	}

	l.size--
}

// moveToFront moves an existing node to the front of the list.
func (l *lruList[K]) moveToFront(node *lruNode[K]) {
	if node == nil || node == l.head {
		return // Already at front or nil
	}

	// Remove from current position
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		// Was tail
		l.tail = node.prev
	}

	// Add to front
	node.prev = nil
	node.next = l.head
	if l.head != nil {
		l.head.prev = node
	}
	l.head = node

	if l.tail == nil {
		l.tail = node
	}
}

// back returns the tail node (least recently used).
func (l *lruList[K]) back() *lruNode[K] {
	return l.tail
}

// front returns the head node (most recently used).
func (l *lruList[K]) front() *lruNode[K] {
	return l.head
}

// len returns the number of nodes in the list.
func (l *lruList[K]) len() int {
	return l.size
}
