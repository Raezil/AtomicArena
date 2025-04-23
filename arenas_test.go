package AtomicArena

import (
	"sync"
	"testing"
)

// TestNewAtomicRingArenaPanics ensures that creating an arena with non-positive size panics.
func TestNewAtomicRingArenaPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for size <= 0, but no panic occurred")
		}
	}()
	_ = NewAtomicArena[int](0)
}

// TestAllocAndPtrAt covers basic allocation, pointer uniqueness, and ring wrap-around.
func TestAllocAndPtrAt(t *testing.T) {
	arena := NewAtomicArena[string](2)

	// First allocation
	p1 := arena.Alloc("first")
	if *p1 != "first" {
		t.Errorf("Alloc returned %q; want %q", *p1, "first")
	}
	// The buffer slot 0 should now hold "first"
	buf0 := arena.PtrAt(0)
	if *buf0 != "first" {
		t.Errorf("PtrAt(0) = %q; want %q", *buf0, "first")
	}

	// Second allocation
	p2 := arena.Alloc("second")
	if *p2 != "second" {
		t.Errorf("Alloc returned %q; want %q", *p2, "second")
	}
	// The buffer slot 1 should now hold "second"
	buf1 := arena.PtrAt(1)
	if *buf1 != "second" {
		t.Errorf("PtrAt(1) = %q; want %q", *buf1, "second")
	}

	// Third allocation wraps around to slot 0
	p3 := arena.Alloc("third")
	if *p3 != "third" {
		t.Errorf("Alloc returned %q; want %q", *p3, "third")
	}
	bufWrap := arena.PtrAt(2)
	if *bufWrap != "third" {
		t.Errorf("PtrAt(2) = %q; want %q", *bufWrap, "third")
	}

	// Ensure Alloc returns independent pointers
	if p1 == buf0 {
		t.Errorf("Alloc pointer and buffer pointer should be different")
	}
}

// TestConcurrentAlloc verifies concurrent allocations produce correct values and unique pointers.
func TestConcurrentAlloc(t *testing.T) {
	const N = 1000
	arena := NewAtomicArena[int](N)
	var wg sync.WaitGroup
	results := make([]*int, N)
	wg.Add(N)

	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			p := arena.Alloc(i)
			results[i] = p
		}(i)
	}
	wg.Wait()

	// Check each value and pointer distinctness
	seen := make(map[*int]bool)
	for i, p := range results {
		if p == nil {
			t.Errorf("Nil pointer at index %d", i)
			continue
		}
		if *p != i {
			t.Errorf("results[%d] = %d; want %d", i, *p, i)
		}
		if seen[p] {
			t.Errorf("Duplicate pointer for value %d", *p)
		}
		seen[p] = true
	}
}

// BenchmarkAlloc measures the performance of Alloc under single-threaded use.
func BenchmarkAlloc(b *testing.B) {
	arena := NewAtomicArena[int](b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arena.Alloc(i)
	}
}

// BenchmarkParallelAlloc measures the performance of Alloc under parallel use.
func BenchmarkParallelAlloc(b *testing.B) {
	arena := NewAtomicArena[int](b.N)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			arena.Alloc(i)
			i++
		}
	})
}
