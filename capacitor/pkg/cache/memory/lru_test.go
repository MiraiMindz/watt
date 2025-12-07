package memory

import "testing"

func TestLRU_Front(t *testing.T) {
	lru := newLRUList[string]()

	// Empty list
	if node := lru.front(); node != nil {
		t.Error("front() on empty list should return nil")
	}

	// Single element
	lru.pushFront("key1")
	if node := lru.front(); node == nil || node.key != "key1" {
		t.Error("front() should return first element")
	}

	// Multiple elements
	lru.pushFront("key2")
	if node := lru.front(); node == nil || node.key != "key2" {
		t.Error("front() should return most recently added element")
	}
}

func TestLRU_Len(t *testing.T) {
	lru := newLRUList[string]()

	// Empty
	if lru.len() != 0 {
		t.Errorf("len() = %d, want 0", lru.len())
	}

	// Add elements
	lru.pushFront("key1")
	if lru.len() != 1 {
		t.Errorf("len() after pushFront = %d, want 1", lru.len())
	}

	lru.pushFront("key2")
	lru.pushFront("key3")
	if lru.len() != 3 {
		t.Errorf("len() = %d, want 3", lru.len())
	}

	// Remove elements
	lru.remove(lru.back())
	if lru.len() != 2 {
		t.Errorf("len() after remove = %d, want 2", lru.len())
	}
}

func TestLRU_RemoveEdgeCases(t *testing.T) {
	lru := newLRUList[string]()

	// Remove from empty list (should not panic)
	lru.remove(nil)

	// Remove single element
	node := lru.pushFront("only")
	lru.remove(node)
	if lru.len() != 0 {
		t.Errorf("len() after removing only element = %d, want 0", lru.len())
	}
	if lru.front() != nil {
		t.Error("front() should be nil after removing only element")
	}
	if lru.back() != nil {
		t.Error("back() should be nil after removing only element")
	}

	// Remove head from multi-element list
	_ = lru.pushFront("a")
	n2 := lru.pushFront("b")
	lru.pushFront("c")
	lru.remove(lru.front())
	if lru.front().key != "b" {
		t.Errorf("front() after removing head = %s, want b", lru.front().key)
	}

	// Remove tail from multi-element list
	lru.remove(lru.back())
	if lru.back().key != "b" {
		t.Errorf("back() after removing tail = %s, want b", lru.back().key)
	}

	// Remove middle element
	lru.pushFront("x")
	lru.pushFront("y")
	lru.remove(n2) // Remove "b" which is now in the middle
	if lru.len() != 2 {
		t.Errorf("len() after removing middle = %d, want 2", lru.len())
	}
}

func TestLRU_MoveToFrontEdgeCases(t *testing.T) {
	lru := newLRUList[string]()

	// Move nil (should not panic)
	lru.moveToFront(nil)

	// Move already at front
	node := lru.pushFront("only")
	lru.moveToFront(node)
	if lru.front() != node {
		t.Error("moveToFront on head should keep it at front")
	}

	// Move from tail
	lru.pushFront("second")
	lru.pushFront("third")
	lru.moveToFront(node) // "only" is now at tail
	if lru.front() != node {
		t.Error("moveToFront from tail should move to front")
	}
	if node.prev != nil {
		t.Error("moved node should have nil prev")
	}

	// Move from middle
	_ = lru.pushFront("a")
	n2 := lru.pushFront("b")
	n3 := lru.pushFront("c")
	lru.moveToFront(n2) // Move "b" from middle
	if lru.front() != n2 {
		t.Error("moveToFront from middle should move to front")
	}
	// Verify list integrity
	if n2.next != n3 || n3.prev != n2 {
		t.Error("list links broken after moveToFront")
	}
}

func TestLRU_PushFrontEmpty(t *testing.T) {
	lru := newLRUList[int]()

	node := lru.pushFront(42)

	if lru.head != node {
		t.Error("head should point to new node")
	}
	if lru.tail != node {
		t.Error("tail should point to new node")
	}
	if node.prev != nil || node.next != nil {
		t.Error("single node should have nil prev and next")
	}
	if lru.len() != 1 {
		t.Errorf("len() = %d, want 1", lru.len())
	}
}

func TestLRU_BackEdgeCases(t *testing.T) {
	lru := newLRUList[string]()

	// Empty list
	if lru.back() != nil {
		t.Error("back() on empty list should return nil")
	}

	// Single element (head == tail)
	node := lru.pushFront("only")
	if lru.back() != node {
		t.Error("back() should equal front() for single element")
	}

	// Multiple elements
	lru.pushFront("newer")
	if lru.back() != node {
		t.Error("back() should return oldest element")
	}
}

func TestLRU_ComplexSequence(t *testing.T) {
	lru := newLRUList[int]()

	// Build a list: 5 -> 4 -> 3 -> 2 -> 1
	nodes := make([]*lruNode[int], 5)
	for i := 0; i < 5; i++ {
		nodes[i] = lru.pushFront(i + 1)
	}

	// Verify order
	if lru.front().key != 5 {
		t.Errorf("front = %d, want 5", lru.front().key)
	}
	if lru.back().key != 1 {
		t.Errorf("back = %d, want 1", lru.back().key)
	}

	// Move middle element to front (3)
	lru.moveToFront(nodes[2])
	if lru.front().key != 3 {
		t.Errorf("front after moveToFront = %d, want 3", lru.front().key)
	}

	// Remove from middle
	lru.remove(nodes[3]) // Remove 2
	if lru.len() != 4 {
		t.Errorf("len after remove = %d, want 4", lru.len())
	}

	// Move tail to front
	lru.moveToFront(lru.back())
	if lru.front().key != 1 {
		t.Errorf("front after moving tail = %d, want 1", lru.front().key)
	}
}
