package managedheap

import (
	"testing"
	"unsafe"
)

// The host fixture owns aligned buffers independently of the collector. Its Go
// maps/slices stand in for OS mappings; they are not target runtime bookkeeping.
type testMemory struct {
	t                                         *testing.T
	buffers                                   map[unsafe.Pointer][]byte
	sizes                                     map[unsafe.Pointer]uintptr
	granule, bytes, peak, ceiling, maxRequest uintptr
	acquires, releases                        int
}

func (m *testMemory) Acquire(size uintptr) unsafe.Pointer {
	m.acquires++
	if m.maxRequest != 0 && size > m.maxRequest {
		return nil
	}
	if size%m.granule != 0 || size > m.ceiling-m.bytes {
		m.t.Fatalf("invalid acquisition: size=%d, held=%d, ceiling=%d", size, m.bytes, m.ceiling)
	}
	buffer := make([]byte, size+m.granule)
	base := unsafe.Pointer(&buffer[0])
	padding := (-uintptr(base)) & (m.granule - 1)
	base = unsafe.Add(base, padding)
	m.buffers[base] = buffer
	m.sizes[base] = size
	m.bytes += size
	if m.bytes > m.peak {
		m.peak = m.bytes
	}
	return base
}

func (m *testMemory) Release(base unsafe.Pointer, size uintptr) {
	if got, ok := m.sizes[base]; !ok || got != size {
		m.t.Fatalf("unmatched release: %p/%d (recorded %d)", base, size, got)
	}
	delete(m.sizes, base)
	delete(m.buffers, base)
	m.bytes -= size
	m.releases++
}

type testRoots struct {
	values  []uintptr
	fail    bool
	calls   int
	reenter bool
	t       *testing.T
}

func (r *testRoots) MarkRoots(h *Heap) bool {
	r.calls++
	for _, pointer := range r.values {
		h.Mark(pointer)
	}
	if r.reenter {
		if h.Alloc(1) != nil || h.Collect() {
			r.t.Fatal("reentrant allocation/collection was accepted")
		}
		h.Close()
	}
	return !r.fail
}

func newTestHeap(t *testing.T, pages uintptr) (*Heap, *testMemory, *testRoots) {
	t.Helper()
	m := &testMemory{t: t, buffers: make(map[unsafe.Pointer][]byte), sizes: make(map[unsafe.Pointer]uintptr), granule: 4096, ceiling: pages * 4096}
	r := &testRoots{t: t}
	h := new(Heap)
	if !h.Init(m, r, Config{Limit: uint64(m.ceiling + unsafe.Sizeof(Heap{})), Granule: m.granule}) {
		t.Fatal("Init failed")
	}
	t.Cleanup(func() {
		h.Close()
		if m.bytes != 0 || len(m.buffers) != 0 {
			t.Fatal("Close leaked segments")
		}
	})
	return h, m, r
}

func alloc(t *testing.T, h *Heap, size uintptr) unsafe.Pointer {
	t.Helper()
	p := h.Alloc(size)
	if p == nil {
		t.Fatalf("Alloc(%d) failed", size)
	}
	if uintptr(p)%alignment != 0 {
		t.Fatalf("unaligned allocation: %p", p)
	}
	return p
}

func alive(h *Heap, p unsafe.Pointer) bool {
	for s := h.first; s != nil; s = s.next {
		for at := segmentSize; at < s.size; {
			b := (*block)(unsafe.Add(unsafe.Pointer(s), at))
			if unsafe.Add(unsafe.Pointer(b), blockSize) == p {
				return b.flags&allocated != 0
			}
			at += b.span
		}
	}
	return false
}

func TestHeapLazyGrowthAndHardLimit(t *testing.T) {
	h, m, r := newTestHeap(t, 3)
	if m.acquires != 0 || h.Reserved() != unsafe.Sizeof(Heap{}) {
		t.Fatal("Init reserved heap segments eagerly")
	}
	a := alloc(t, h, 3000)
	*(*uintptr)(a) = 17
	r.values = append(r.values, uintptr(a))
	b := alloc(t, h, 6000)
	r.values = append(r.values, uintptr(b))
	if *(*uintptr)(a) != 17 || h.Reserved() != m.ceiling+unsafe.Sizeof(Heap{}) {
		t.Fatal("growth moved/corrupted live storage or miscounted budget")
	}
	calls, acquires := r.calls, m.acquires
	if h.Alloc(6000) != nil || r.calls != calls+1 || m.acquires != acquires {
		t.Fatal("limit failure must collect without acquiring over budget")
	}
	if m.peak != m.ceiling {
		t.Fatal("peak differs from exact segment budget")
	}
}

func TestHeapInteriorRootsTransitiveCyclesAndRelease(t *testing.T) {
	h, m, r := newTestHeap(t, 4)
	a := alloc(t, h, 64)
	b := alloc(t, h, 64)
	c := alloc(t, h, 64)
	dead := alloc(t, h, 64)
	*(*uintptr)(a) = uintptr(b) + 7
	*(*uintptr)(b) = uintptr(c) + 63
	*(*uintptr)(c) = uintptr(a)
	r.values = []uintptr{uintptr(c) + 1, 1, ^uintptr(0)}
	if !h.Collect() || !alive(h, a) || !alive(h, b) || !alive(h, c) || alive(h, dead) {
		t.Fatal("interior root, backward transitive edge, cycle or garbage handling failed")
	}
	r.values = nil
	if !h.Collect() || m.bytes != 0 || h.Reserved() != unsafe.Sizeof(Heap{}) {
		t.Fatal("unreachable cycle/empty segment was not released")
	}
}

func TestHeapCoalescesAndZeroesReusedBlocks(t *testing.T) {
	h, m, r := newTestHeap(t, 1)
	a := alloc(t, h, 64)
	b := alloc(t, h, 64)
	c := alloc(t, h, 64)
	for i := uintptr(0); i < 64; i++ {
		*(*byte)(unsafe.Add(a, i)) = 0xff
		*(*byte)(unsafe.Add(b, i)) = 0xff
	}
	r.values = []uintptr{uintptr(c)}
	if !h.Collect() {
		t.Fatal("collection failed")
	}
	before := m.acquires
	reused := alloc(t, h, 128)
	if reused != a || m.acquires != before {
		t.Fatal("adjacent garbage was not coalesced/reused")
	}
	for i := uintptr(0); i < 128; i++ {
		if *(*byte)(unsafe.Add(reused, i)) != 0 {
			t.Fatalf("stale payload or absorbed header at byte %d", i)
		}
	}
}

func TestHeapIncompleteRootsPreserveEverything(t *testing.T) {
	h, m, r := newTestHeap(t, 1)
	a := alloc(t, h, 64)
	b := alloc(t, h, 64)
	r.values = []uintptr{uintptr(a)}
	r.fail = true
	if h.Collect() || !alive(h, a) || !alive(h, b) || m.releases != 0 {
		t.Fatal("incomplete enumeration reclaimed memory")
	}
	if h.Alloc(4096) != nil || m.acquires != 1 {
		t.Fatal("failed collection must not proceed to growth")
	}
	r.fail = false
	r.values = nil
	if !h.Collect() || m.bytes != 0 {
		t.Fatal("aborted marks leaked into the next collection")
	}
}

func TestHeapRepeatedPressurePreservesLiveData(t *testing.T) {
	h, m, r := newTestHeap(t, 2)
	keeper := alloc(t, h, 64)
	*(*uintptr)(keeper) = 123
	r.values = []uintptr{uintptr(keeper)}
	for i := 0; i < 1000; i++ {
		p := alloc(t, h, 700)
		*(*uintptr)(p) = 77
		if *(*uintptr)(keeper) != 123 {
			t.Fatalf("live data damaged at iteration %d", i)
		}
	}
	if r.calls == 0 || m.acquires != 1 {
		t.Fatal("allocation pressure did not reclaim/reuse existing storage")
	}
}

func TestHeapHostFailureFallbackAndNoAccountingLeak(t *testing.T) {
	h, m, r := newTestHeap(t, 4)
	m.maxRequest = 4096
	a := alloc(t, h, 3000)
	r.values = []uintptr{uintptr(a)}
	b := alloc(t, h, 3000) // preferred 8192 fails; minimum 4096 succeeds
	r.values = append(r.values, uintptr(b))
	if m.acquires != 3 || m.bytes != 8192 {
		t.Fatal("did not retry failed preferred growth at minimum size")
	}
	before := h.Reserved()
	if h.Alloc(5000) != nil || h.Reserved() != before {
		t.Fatal("host failure leaked budget or produced an allocation")
	}
	if !alive(h, a) || !alive(h, b) {
		t.Fatal("host failure lost live objects")
	}
}

func TestHeapRejectsOverflowAndReentrancy(t *testing.T) {
	h, m, r := newTestHeap(t, 1)
	for _, size := range []uintptr{^uintptr(0), ^uintptr(0) - alignment, ^uintptr(0) - 4095} {
		if h.Alloc(size) != nil || m.acquires != 0 {
			t.Fatal("overflowing allocation reached provider")
		}
	}
	a := alloc(t, h, 0)
	r.values = []uintptr{uintptr(a)}
	r.reenter = true
	if !h.Collect() || !alive(h, a) {
		t.Fatal("reentrant requests disrupted collection")
	}
	if h.Init(m, r, Config{Granule: 4096}) {
		t.Fatal("reinitialized a live heap")
	}
}

func TestHeapCollectionDoesNotAllocate(t *testing.T) {
	h, _, r := newTestHeap(t, 1)
	a := alloc(t, h, 64)
	r.values = []uintptr{uintptr(a)}
	if allocations := testing.AllocsPerRun(100, func() {
		if !h.Collect() {
			t.Fatal("collection failed")
		}
	}); allocations != 0 {
		t.Fatalf("collector allocated %g objects per pass", allocations)
	}
}

func TestHeapInvalidConfigurationAndBookkeeping(t *testing.T) {
	_, m, r := newTestHeap(t, 1)
	for _, config := range []Config{
		{Granule: 0}, {Granule: 3}, {Granule: 16},
		{Granule: 4096, Limit: 1},
		{Granule: 4096, Overhead: ^uintptr(0)},
		{Granule: 4096, Initial: ^uintptr(0)},
	} {
		var h Heap
		if h.Init(m, r, config) || h.Reserved() != 0 {
			t.Fatalf("invalid config accepted or changed heap: %+v", config)
		}
	}
	var h Heap
	if h.Init(m, nil, Config{Granule: 4096}) || h.Init(nil, r, Config{Granule: 4096}) {
		t.Fatal("missing provider/root enumerator accepted")
	}
	if !h.Init(m, r, Config{Granule: 4096, Overhead: 512, Limit: uint64(unsafe.Sizeof(Heap{}) + 512 + 4095)}) {
		t.Fatal("valid small budget rejected")
	}
	if h.Alloc(1) != nil || m.acquires != 0 {
		t.Fatal("fixed bookkeeping was not charged before growth")
	}
	h.Close()
}
