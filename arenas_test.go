package atomicarena

import (
	"sync"
	"testing"
)

// TestAllocInt ensures Alloc succeeds when within capacity for int
func TestAllocInt(t *testing.T) {
	// Create arena for exactly one int (size 8 bytes on 64-bit)
	arena := NewAtomicArena[int](1)
	val, err := arena.Alloc(42)
	if err != nil {
		t.Fatalf("Alloc returned error: %v", err)
	}
	if *val != 42 {
		t.Errorf("expected 42, got %d", *val)
	}
}

// TestAllocStruct ensures Alloc succeeds for a struct and errors when exceeding capacity
func TestAllocStruct(t *testing.T) {
	type S struct{ A, B int64 }
	// size of S is 16 bytes, so maxElems=1 gives capacity 16
	arena := NewAtomicArena[S](1)
	_, err := arena.Alloc(S{1, 2})
	if err != nil {
		t.Fatalf("Alloc returned error: %v", err)
	}
	// Second alloc should fail
	_, err = arena.Alloc(S{3, 4})
	if err == nil {
		t.Fatal("expected error on second alloc, got none")
	}
}

// TestReset ensures Reset allows reusing arena after clearing
func TestReset(t *testing.T) {
	arena := NewAtomicArena[int](1)
	_, err := arena.Alloc(7)
	if err != nil {
		t.Fatalf("initial alloc failed: %v", err)
	}
	arena.Reset()
	// After reset, should be able to alloc again
	_, err = arena.Alloc(8)
	if err != nil {
		t.Fatalf("Alloc after reset failed: %v", err)
	}
}

// TestConcurrentAlloc tests concurrent allocations up to capacity
func TestConcurrentAlloc(t *testing.T) {
	arena := NewAtomicArena[int](10)
	count := 10

	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := arena.Alloc(i)
			if err != nil {
				t.Errorf("unexpected error on alloc %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	_, err := arena.Alloc(99)
	if err == nil {
		t.Fatal("expected error after capacity reached, got none")
	}
}
