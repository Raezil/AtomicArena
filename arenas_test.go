package atomicarena

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewAtomicArena(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"valid size", 10, false},
		{"zero size", 0, true},
		{"negative size", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arena, err := NewAtomicArena[int](tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAtomicArena() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && arena == nil {
				t.Errorf("NewAtomicArena() returned nil arena without error")
			}
			if !tt.wantErr && arena.size != uint64(tt.size) {
				t.Errorf("NewAtomicArena() incorrect size, got %v, want %v", arena.size, tt.size)
			}
		})
	}
}

func TestAtomicArena_Alloc(t *testing.T) {
	t.Run("basic allocation", func(t *testing.T) {
		arena, err := NewAtomicArena[string](3)
		if err != nil {
			t.Fatalf("Failed to create arena: %v", err)
		}

		values := []string{"first", "second", "third"}
		for i, val := range values {
			ptr, err := arena.Alloc(val)
			if err != nil {
				t.Errorf("Alloc(%q) error = %v", val, err)
			}
			if *ptr != val {
				t.Errorf("Alloc(%q) = %q, want %q", val, *ptr, val)
			}
			if stored := arena.PtrAt(uint64(i)); stored == nil || *stored != val {
				t.Errorf("PtrAt(%d) = %v, want %q", i, stored, val)
			}
		}

		// Should be full now
		_, err = arena.Alloc("fourth")
		if err == nil {
			t.Error("Expected error when arena is full, got nil")
		}
	})
}

func TestAtomicArena_Reset(t *testing.T) {
	arena, _ := NewAtomicArena[int](3)

	// Allocate some values
	for i := 0; i < 3; i++ {
		_, err := arena.Alloc(i)
		if err != nil {
			t.Fatalf("Failed to allocate: %v", err)
		}
	}

	// Arena should be full
	if arena.counter != 3 {
		t.Errorf("Counter = %d, want 3", arena.counter)
	}

	// Reset the arena
	arena.Reset()

	// Counter should be reset
	if arena.counter != 0 {
		t.Errorf("Counter after reset = %d, want 0", arena.counter)
	}

	// Slots should be nil
	for i := 0; i < 3; i++ {
		if ptr := arena.PtrAt(uint64(i)); ptr != nil {
			t.Errorf("PtrAt(%d) = %v, want nil", i, ptr)
		}
	}

	// Should be able to allocate again
	for i := 0; i < 3; i++ {
		_, err := arena.Alloc(i)
		if err != nil {
			t.Errorf("Failed to allocate after reset: %v", err)
		}
	}
}

func TestAtomicArena_PtrAt(t *testing.T) {
	t.Run("modulo behavior", func(t *testing.T) {
		arena, _ := NewAtomicArena[int](3)

		// Allocate values
		for i := 0; i < 3; i++ {
			_, err := arena.Alloc(i * 10)
			if err != nil {
				t.Fatalf("Failed to allocate: %v", err)
			}
		}

		// Test that PtrAt uses modulo
		tests := []struct {
			index uint64
			want  int
		}{
			{0, 0},
			{1, 10},
			{2, 20},
			{3, 0},  // should wrap around to index 0
			{4, 10}, // should wrap around to index 1
			{5, 20}, // should wrap around to index 2
		}

		for _, tt := range tests {
			ptr := arena.PtrAt(tt.index)
			if ptr == nil {
				t.Errorf("PtrAt(%d) = nil, want %d", tt.index, tt.want)
			} else if *ptr != tt.want {
				t.Errorf("PtrAt(%d) = %d, want %d", tt.index, *ptr, tt.want)
			}
		}
	})

	t.Run("empty arena", func(t *testing.T) {
		var arena AtomicArena[int]
		if ptr := arena.PtrAt(0); ptr != nil {
			t.Errorf("PtrAt(0) for empty arena = %v, want nil", ptr)
		}
	})
}

func TestAtomicArena_ConcurrentAlloc(t *testing.T) {
	const (
		numGoroutines      = 10
		allocsPerGoroutine = 100
	)

	arena, _ := NewAtomicArena[int](numGoroutines * allocsPerGoroutine)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start goroutines that concurrently allocate
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			base := goroutineID * allocsPerGoroutine
			for i := 0; i < allocsPerGoroutine; i++ {
				val := base + i
				ptr, err := arena.Alloc(val)
				if err != nil {
					t.Errorf("Goroutine %d: Alloc(%d) error = %v", goroutineID, val, err)
					return
				}
				if *ptr != val {
					t.Errorf("Goroutine %d: Alloc(%d) = %d, want %d", goroutineID, val, *ptr, val)
				}
			}
		}(g)
	}

	wg.Wait()

	// Verify all values are allocated and accessible
	allocated := make(map[int]bool)
	for i := uint64(0); i < arena.size; i++ {
		ptr := arena.PtrAt(i)
		if ptr == nil {
			t.Errorf("PtrAt(%d) = nil, expected a value", i)
			continue
		}
		val := *ptr
		if allocated[val] {
			t.Errorf("Value %d was allocated multiple times", val)
		}
		allocated[val] = true
	}

	if len(allocated) != numGoroutines*allocsPerGoroutine {
		t.Errorf("Found %d allocated values, want %d", len(allocated), numGoroutines*allocsPerGoroutine)
	}
}

// Benchmark the AtomicArena

func BenchmarkAtomicArena_Alloc(b *testing.B) {
	arena, _ := NewAtomicArena[int](b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := arena.Alloc(i)
		if err != nil {
			b.Fatalf("Failed to allocate: %v", err)
		}
	}
}

func BenchmarkAtomicArena_PtrAt(b *testing.B) {
	const size = 1000
	arena, _ := NewAtomicArena[int](size)

	// Fill the arena
	for i := 0; i < size; i++ {
		_, _ = arena.Alloc(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = arena.PtrAt(uint64(i % size))
	}
}

func BenchmarkAtomicArena_Reset(b *testing.B) {
	const size = 1000
	arena, _ := NewAtomicArena[int](size)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Fill the arena
		for j := 0; j < size; j++ {
			_, _ = arena.Alloc(j)
		}
		b.StartTimer()

		// Benchmark the reset operation
		arena.Reset()
	}
}

func BenchmarkAtomicArena_ConcurrentAlloc(b *testing.B) {
	for _, numGoroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("Goroutines-%d", numGoroutines), func(b *testing.B) {
			allocsPerGoroutine := b.N / numGoroutines
			if allocsPerGoroutine == 0 {
				allocsPerGoroutine = 1
			}
			totalAllocs := allocsPerGoroutine * numGoroutines

			arena, _ := NewAtomicArena[int](totalAllocs)

			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			b.ResetTimer()
			for g := 0; g < numGoroutines; g++ {
				go func(goroutineID int) {
					defer wg.Done()
					base := goroutineID * allocsPerGoroutine
					for i := 0; i < allocsPerGoroutine; i++ {
						val := base + i
						_, _ = arena.Alloc(val)
					}
				}(g)
			}

			wg.Wait()
		})
	}
}

func TestAllocSlice_Success(t *testing.T) {
	a, _ := NewAtomicArena[int](3)
	vals := []int{1, 2, 3}
	ptrs, err := a.AllocSlice(vals)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	for i, v := range vals {
		if *ptrs[i] != v {
			t.Errorf("ptrs[%d]=%d; want %d", i, *ptrs[i], v)
		}
	}
}

func TestAllocSlice_Overflow(t *testing.T) {
	a, _ := NewAtomicArena[int](2)
	_, err := a.AllocSlice([]int{1, 2, 3})
	if err == nil {
		t.Errorf("expected overflow error, got nil")
	}
}
func TestMakeSlice_Full(t *testing.T) {
	a, _ := NewAtomicArena[int](3)
	vals := []int{1, 2, 3}
	a.AllocSlice(vals)
	slice, err := a.MakeSlice()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	for i, v := range vals {
		if *slice[i] != v {
			t.Errorf("slice[%d]=%d; want %d", i, *slice[i], v)
		}
	}
}

func BenchmarkAllocSlice(b *testing.B) {
	a, _ := NewAtomicArena[int](1000)
	vals := make([]int, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Reset()
		a.AllocSlice(vals)
	}
}

func BenchmarkMakeSlice(b *testing.B) {
	a, _ := NewAtomicArena[int](1000)
	vals := make([]int, 1000)
	a.AllocSlice(vals)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.MakeSlice()
	}
}

func TestAppendSlice(t *testing.T) {
	arena, err := NewAtomicArena[int](5)
	if err != nil {
		t.Fatalf("failed to create arena: %v", err)
	}
	// Pre-allocate some values
	dest, err := arena.AllocSlice([]int{1, 2})
	if err != nil {
		t.Fatalf("AllocSlice failed: %v", err)
	}
	dest, err = arena.AppendSlice(dest, []int{3, 4})
	if err != nil {
		t.Fatalf("AppendSlice failed: %v", err)
	}

	// Verify length and contents
	if len(dest) != 4 {
		t.Errorf("expected length 4, got %d", len(dest))
	}
	for i, ptr := range dest {
		want := i + 1
		if *ptr != want {
			t.Errorf("dest[%d]=%d, want %d", i, *ptr, want)
		}
	}
}

func TestAppendSlice_Oversize(t *testing.T) {
	arena, err := NewAtomicArena[int](3)
	if err != nil {
		t.Fatalf("failed to create arena: %v", err)
	}
	// Allocate one slot
	dest, err := arena.AllocSlice([]int{1})
	if err != nil {
		t.Fatalf("AllocSlice failed: %v", err)
	}
	// Try to append too many values
	_, err = arena.AppendSlice(dest, []int{2, 3, 4})
	if err == nil {
		t.Fatal("expected error when appending beyond capacity, got nil")
	}
}

func BenchmarkAppendSlice(b *testing.B) {
	for _, size := range []int{8, 1024, 1 << 20} {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			arena, err := NewAtomicArena[int](size * 2)
			if err != nil {
				b.Fatalf("failed to create arena: %v", err)
			}
			vals := make([]int, size)
			for i := range vals {
				vals[i] = i
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dest := make([]*int, 0)
				dest, _ = arena.AppendSlice(dest, vals)
			}
		})
	}
}

type dummyValue = int

func BenchmarkAtomicArenaAlloc(b *testing.B) {
	const N = 1000
	a, err := NewAtomicArena[dummyValue](N)
	if err != nil {
		b.Fatalf("failed to create arena: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// reset when full to avoid ErrArenaFull
		if _, err := a.Alloc(dummyValue(i)); err != nil {
			a.Reset()
			_, _ = a.Alloc(dummyValue(i))
		}
	}
}

func BenchmarkNative_New(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = new(dummyValue)
	}
}

func BenchmarkAtomicArena_AllocSlice(b *testing.B) {
	const N = 1000
	a, err := NewAtomicArena[dummyValue](N)
	if err != nil {
		b.Fatalf("failed to create arena: %v", err)
	}
	vals := make([]dummyValue, N)
	for i := range vals {
		vals[i] = dummyValue(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Reset()
		if _, err := a.AllocSlice(vals); err != nil {
			b.Fatalf("AllocSlice error: %v", err)
		}
	}
}

func BenchmarkNative_SliceAlloc(b *testing.B) {
	const N = 1000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice := make([]*dummyValue, 0, N)
		for j := 0; j < N; j++ {
			slice = append(slice, new(dummyValue))
		}
		_ = slice
	}
}

const arenaSize = 1000000

type S100 [100]byte
type S1000 [1000]byte
type S10000 [10000]byte
type S100000 [100000]byte
type S1000000 [1000000]byte

func BenchmarkS100(b *testing.B) {
	b.Run("Native", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = new(S100)
		}
	})
	b.Run("Arena", func(b *testing.B) {
		b.ReportAllocs()
		arena, _ := NewAtomicArena[S100](arenaSize)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			arena.Alloc(S100{})
		}
	})
}

func BenchmarkS1000(b *testing.B) {
	b.Run("Native", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = new(S1000)
		}
	})
	b.Run("Arena", func(b *testing.B) {
		b.ReportAllocs()
		arena, _ := NewAtomicArena[S1000](arenaSize)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			arena.Alloc(S1000{})
		}
	})
}

func BenchmarkS10000(b *testing.B) {
	b.Run("Native", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = new(S10000)
		}
	})
	b.Run("Arena", func(b *testing.B) {
		b.ReportAllocs()
		arena, _ := NewAtomicArena[S10000](arenaSize)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			arena.Alloc(S10000{})
		}
	})
}

func BenchmarkS100000(b *testing.B) {
	b.Run("Native", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = new(S100000)
		}
	})
	b.Run("Arena", func(b *testing.B) {
		b.ReportAllocs()
		arena, _ := NewAtomicArena[S100000](arenaSize)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			arena.Alloc(S100000{})
		}
	})
}

func BenchmarkS1000000(b *testing.B) {
	b.Run("Native", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = new(S1000000)
		}
	})
	b.Run("Arena", func(b *testing.B) {
		b.ReportAllocs()
		arena, _ := NewAtomicArena[S1000000](arenaSize)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			arena.Alloc(S1000000{})
		}
	})
}
