package atomicarena

import (
	"testing"
	"unsafe"
)

type MyStruct struct {
	A, B int64
}

type LargeArray [1024]byte

type HugeArray [1 << 20]byte // ~1MB

func TestAllocInt(t *testing.T) {
	mem := &AtomicArena[int]{}
	const N = 5
	var sumOffsets uintptr

	for i := 0; i < N; i++ {
		ptr, err := mem.Alloc(i * 10)
		if err != nil {
			t.Fatalf("Alloc returned error: %v", err)
		}
		if ptr == nil {
			t.Fatal("Alloc returned nil pointer")
		}
		if *ptr != i*10 {
			t.Errorf("got %d; want %d", *ptr, i*10)
		}
		sumOffsets += unsafe.Sizeof(i)
	}

	if len(mem.buff) != N {
		t.Errorf("buff length = %d; want %d", len(mem.buff), N)
	}
	if mem.offset != sumOffsets {
		t.Errorf("offset = %d; want %d", mem.offset, sumOffsets)
	}
}

func TestAllocStruct(t *testing.T) {
	mem := &AtomicArena[MyStruct]{}
	val := MyStruct{A: 42, B: 99}
	ptr, err := mem.Alloc(val)
	if err != nil {
		t.Fatalf("Alloc returned error: %v", err)
	}
	if ptr == nil {
		t.Fatal("Alloc returned nil pointer")
	}
	if *ptr != val {
		t.Errorf("got %+v; want %+v", *ptr, val)
	}

	expectedOffset := unsafe.Sizeof(val)
	if mem.offset != expectedOffset {
		t.Errorf("offset = %d; want %d", mem.offset, expectedOffset)
	}
}

func TestReset(t *testing.T) {
	mem := &AtomicArena[int]{}
	// allocate a few items
	for i := 0; i < 3; i++ {
		if _, err := mem.Alloc(i); err != nil {
			t.Fatalf("Alloc failed: %v", err)
		}
	}
	// now reset
	mem.Reset()
	if mem.offset != 0 {
		t.Errorf("after Reset, offset = %d; want 0", mem.offset)
	}
	for i, ap := range mem.buff {
		if got := ap.Load(); got != nil {
			t.Errorf("buff[%d].Load() = %v; want nil", i, got)
		}
	}
}

// Benchmarks for int allocations
func BenchmarkArenaAllocInt(b *testing.B) {
	mem := &AtomicArena[int]{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.Alloc(i)
	}
}

func BenchmarkNewInt(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ptr := new(int)
		*ptr = i
		_ = ptr
	}
}

// Benchmarks for ~1KB allocations
func BenchmarkArenaAllocLargeArray(b *testing.B) {
	mem := &AtomicArena[LargeArray]{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.Alloc(LargeArray{})
	}
}

func BenchmarkNewLargeArray(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ptr := new(LargeArray)
		_ = ptr
	}
}

// Benchmarks for ~1MB allocations
func BenchmarkArenaAllocHugeArray(b *testing.B) {
	mem := &AtomicArena[HugeArray]{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.Alloc(HugeArray{})
	}
}

func BenchmarkNewHugeArray(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ptr := new(HugeArray)
		_ = ptr
	}
}
