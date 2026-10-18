package main

import (
	"renvo.dev/internal/managedheap"
	"unsafe"
)

// This fixture supplies static storage and explicit roots. It tests compiled
// core semantics, not OS acquisition or automatic root discovery.
var backing [32768]byte

type memory struct{ used uintptr }

func (m *memory) Acquire(size uintptr) unsafe.Pointer {
	base := unsafe.Pointer(&backing[0])
	start := ((uintptr(base) + m.used + 4095) &^ uintptr(4095)) - uintptr(base)
	if start+size > uintptr(len(backing)) {
		return nil
	}
	m.used = start + size
	return unsafe.Add(base, start)
}
func (m *memory) Release(base unsafe.Pointer, size uintptr) {}

type roots struct{ root uintptr }

func (r *roots) MarkRoots(h *managedheap.Heap) bool {
	h.Mark(r.root)
	return true
}

func main() {
	var m memory
	var r roots
	var h managedheap.Heap
	limit, ok := managedheap.ResolveLimit(0, 8192, "12KiB", true)
	if !ok || limit != 12288 {
		panic("limit")
	}
	if !h.Init(&m, &r, managedheap.Config{Limit: limit, Granule: 4096}) {
		panic("init")
	}
	a := h.Alloc(64)
	b := h.Alloc(64)
	if a == nil || b == nil {
		panic("allocation")
	}
	*(*uintptr)(a) = uintptr(b) + 1
	*(*uintptr)(b) = 123
	r.root = uintptr(a) + 1
	for i := 0; i < 100; i++ {
		if h.Alloc(700) == nil {
			panic("pressure")
		}
		if *(*uintptr)(b) != 123 {
			panic("live data")
		}
	}
	if !h.Collect() || *(*uintptr)(b) != 123 {
		panic("collect")
	}
	if h.Alloc(12288) != nil {
		panic("limit bypass")
	}
	r.root = 0
	if !h.Collect() || h.Reserved() != unsafe.Sizeof(managedheap.Heap{}) {
		panic("release")
	}
	h.Close()
	var uncapped managedheap.Heap
	if !uncapped.Init(&m, &r, managedheap.Config{Granule: 4096}) {
		panic("uncapped init")
	}
	if uncapped.Alloc(^uintptr(0)) != nil {
		panic("overflow")
	}
	if _, valid := managedheap.ParseLimit("18446744073709551616"); valid {
		panic("parse overflow")
	}
	uncapped.Close()
	println("PASS")
}
