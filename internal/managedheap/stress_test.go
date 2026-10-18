package managedheap

import (
	"testing"
	"unsafe"
)

// Mutation, dropping roots, failed enumeration and reallocation all occur in
// one heap. The oracle follows logical object IDs, never allocator metadata.
func TestHeapStatefulGraphStress(t *testing.T) {
	for seed := uint32(1); seed <= 32; seed++ {
		h, _, roots := newTestHeap(t, 1)
		var objects [48]unsafe.Pointer
		var edges [48][2]int
		random := seed
		next := func() int { random = random*1664525 + 1013904223; return int(random % 48) }
		for step := 0; step < 128; step++ {
			for i := range objects {
				if objects[i] != nil {
					continue
				}
				objects[i] = alloc(t, h, 64)
				for j := uintptr(0); j < 64; j++ {
					if *(*byte)(unsafe.Add(objects[i], j)) != 0 {
						t.Fatalf("seed %d step %d: dirty reuse", seed, step)
					}
				}
				edges[i] = [2]int{-1, -1}
			}
			for mutation := 0; mutation < 12; mutation++ {
				from, to := next(), next()
				side := mutation % 2
				edges[from][side] = to
				*(*uintptr)(unsafe.Add(objects[from], uintptr(side)*wordSize)) = uintptr(objects[to]) + 3
			}
			roots.values = nil
			var queue []int
			for i := 0; i < 4; i++ {
				id := next()
				queue = append(queue, id)
				roots.values = append(roots.values, uintptr(objects[id])+31)
			}
			var reachable [48]bool
			for len(queue) > 0 {
				id := queue[len(queue)-1]
				queue = queue[:len(queue)-1]
				if id < 0 || reachable[id] {
					continue
				}
				reachable[id] = true
				queue = append(queue, edges[id][0], edges[id][1])
			}
			if step%7 == 0 {
				roots.fail = true
				if h.Collect() {
					t.Fatal("accepted incomplete roots")
				}
				roots.fail = false
				for _, p := range objects {
					if !alive(h, p) {
						t.Fatal("failed enumeration reclaimed an object")
					}
				}
			}
			if !h.Collect() {
				t.Fatal("collection failed")
			}
			for i, p := range objects {
				if alive(h, p) != reachable[i] {
					t.Fatalf("seed %d step %d object %d: reachability mismatch", seed, step, i)
				}
				if !reachable[i] {
					objects[i] = nil
				}
			}
		}
		roots.values = nil
		if !h.Collect() || h.Reserved() != unsafe.Sizeof(Heap{}) {
			t.Fatal("final graph leaked")
		}
		h.Close()
	}
}

func TestHeapFragmentationAtHardLimit(t *testing.T) {
	h, m, roots := newTestHeap(t, 1)
	a := alloc(t, h, 1000)
	roots.values = []uintptr{uintptr(a)}
	b := alloc(t, h, 1000)
	roots.values = append(roots.values, uintptr(b))
	c := alloc(t, h, 1000)
	roots.values = append(roots.values, uintptr(c))
	roots.values = []uintptr{uintptr(b)}
	if !h.Collect() {
		t.Fatal("collection")
	}
	before := m.acquires
	// Aggregate free payload exceeds 2500, but the middle live block separates
	// it into holes. A nonmoving collector must fail rather than exceed the cap.
	if h.Alloc(2500) != nil || m.acquires != before {
		t.Fatal("fragmented request exceeded budget")
	}
	if !alive(h, b) {
		t.Fatal("fragmentation reclaimed live anchor")
	}
	roots.values = nil
	if !h.Collect() {
		t.Fatal("release anchor")
	}
	if alloc(t, h, 2500) == nil {
		t.Fatal("allocation remained poisoned after OOM")
	}
}

func TestHeapConservativeIntegerRetention(t *testing.T) {
	h, _, roots := newTestHeap(t, 1)
	payload := uintptr(4096) - segmentSize - blockSize
	p := alloc(t, h, payload)
	// A pointer-shaped integer is indistinguishable from a real reference.
	roots.values = []uintptr{uintptr(p) + payload - 1}
	if h.Alloc(64) != nil {
		t.Fatal("ignored conservative interior root")
	}
	if !alive(h, p) {
		t.Fatal("lost conservatively retained allocation")
	}
	roots.values = nil
	if !h.Collect() {
		t.Fatal("collection")
	}
	alloc(t, h, 64)
}
