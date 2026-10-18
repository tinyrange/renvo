package managedheap

import (
	"testing"
	"unsafe"
)

// Compare the collector with an independent graph traversal. Two disconnected
// subgraphs ensure each run has garbage even when the rooted subgraph is cyclic.
func TestHeapReachabilityAgainstGraphOracle(t *testing.T) {
	for seed := uint32(1); seed <= 32; seed++ {
		h, _, roots := newTestHeap(t, 1)
		var objects [48]unsafe.Pointer
		var edges [48][2]int
		random := seed
		for i := range objects {
			objects[i] = alloc(t, h, 2*wordSize)
		}
		for i := range objects {
			for j := 0; j < 2; j++ {
				random = random*1664525 + 1013904223
				next := i/24*24 + int(random%24)
				edges[i][j] = next
				// The odd offset exercises interior pointers, not just bases.
				*(*uintptr)(unsafe.Add(objects[i], uintptr(j)*wordSize)) = uintptr(objects[next]) + 1
			}
		}
		root := int(seed % 24)
		roots.values = []uintptr{uintptr(objects[root]) + wordSize - 1}
		var reachable [48]bool
		queue := []int{root}
		for len(queue) != 0 {
			i := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			if reachable[i] {
				continue
			}
			reachable[i] = true
			queue = append(queue, edges[i][0], edges[i][1])
		}
		if !h.Collect() {
			t.Fatal("collection failed")
		}
		for i, object := range objects {
			if alive(h, object) != reachable[i] {
				t.Fatalf("seed %d object %d: alive=%v, reachable=%v", seed, i, alive(h, object), reachable[i])
			}
		}
		h.Close()
	}
}

func TestHeapUnlinksEmptySegmentsAroundLiveSegment(t *testing.T) {
	h, m, roots := newTestHeap(t, 7)
	a := alloc(t, h, 3000)
	roots.values = []uintptr{uintptr(a)}
	b := alloc(t, h, 6000)
	roots.values = append(roots.values, uintptr(b))
	c := alloc(t, h, 12000)
	roots.values = []uintptr{uintptr(b)}
	*(*uintptr)(b) = 99
	if !h.Collect() || m.bytes != 8192 || m.releases != 2 {
		t.Fatal("did not release both ends of segment list")
	}
	if alive(h, a) || !alive(h, b) || alive(h, c) || *(*uintptr)(b) != 99 {
		t.Fatal("unlinked the wrong segment")
	}
	roots.values = nil
	if !h.Collect() || m.bytes != 0 {
		t.Fatal("final segment leaked")
	}
}
