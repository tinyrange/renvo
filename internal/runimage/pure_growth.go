//go:build !renvo

package runimage

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"unsafe"
)

// Growable code keeps immutable entry offsets stable across relocation. Calls,
// pin leases, native target records and physical code publication share mu.
func (a *CodeArena) AllowGrowth() { a.mu.Lock(); a.growable = true; a.mu.Unlock() }
func (a *CodeArena) growLocked(required int) error {
	if a.base == 0 || a.broken || required < 1 || uint64(required) > uint64(^uint32(0)) {
		return fmt.Errorf("native code offset exhausted")
	}
	size := a.size
	for size < required {
		if uint64(size)*2 > uint64(^uint32(0)) {
			size = (required + 16383) &^ 16383
			break
		}
		size *= 2
	}
	base, err := pureMap(size)
	if err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	a.releaseOwnedPinsLocked()
	for _, extent := range a.extents {
		if extent.length == 0 || a.entries[extent.offset>>4].Words == 0 {
			continue
		}
		copy(unsafe.Slice((*byte)(unsafe.Pointer(base+uintptr(extent.offset))), int(extent.length)), unsafe.Slice((*byte)(unsafe.Pointer(a.base+uintptr(extent.offset))), int(extent.length)))
	}
	page := os.Getpagesize()
	published := (a.used + page - 1) &^ (page - 1)
	if published != 0 {
		if err = pureSeal(base, published, 0, a.used); err != nil {
			pureUnmap(base, size)
			return err
		}
	}
	old := a.base
	if err = pureUnmap(old, a.size); err != nil {
		pureUnmap(base, size)
		return err
	}
	a.base, a.size = base, size
	for entry, name := range a.named {
		at := sort.Search(len(a.extents), func(i int) bool { return a.extents[i].offset >= uint32(entry) })
		if at < len(a.extents) && a.extents[at].offset == uint32(entry) && a.extents[at].length != 0 {
			if err := a.publishSymbol(base+uintptr(entry), int(a.extents[at].length), name); err != nil {
				a.symbolError = err
			}
		}
	}
	for _, call := range a.prepared {
		call.entry = base + (call.entry - old)
	}
	for i := range a.faults {
		a.faults[i].PC = base + (a.faults[i].PC - old)
		a.faults[i].Recovery = base + (a.faults[i].Recovery - old)
	}
	a.linkedView = [4]uint64{uint64(base), uint64(a.used), uint64(uintptr(unsafe.Pointer(&a.entries[0]))), uint64(len(a.entries))}
	return nil
}

// Release revokes admission first. Offsets are never reused, so stale public
// descriptors cannot gain authority over a different fragment later. Entire
// retired host pages are discarded without touching live neighboring images.
func (a *CodeArena) Release(entry int) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.base == 0 || entry < 0 || entry&15 != 0 || entry>>4 >= len(a.entries) || a.entries[entry>>4].Words == 0 {
		return 0
	}
	at := sort.Search(len(a.extents), func(i int) bool { return a.extents[i].offset >= uint32(entry) })
	if at == len(a.extents) || a.extents[at].offset != uint32(entry) {
		return 0
	}
	extent := a.extents[at]
	a.extents[at].length = 0
	a.entries[entry>>4] = arenaEntry{}
	for i := range a.targets {
		if a.targets[i].Entry == uint64(entry) {
			a.targets[i] = linkedTarget{}
		}
	}
	first, last := a.base+uintptr(entry), a.base+uintptr(entry)+uintptr(extent.length)
	faults := a.faults[:0]
	for _, site := range a.faults {
		if site.PC < first || site.PC >= last {
			faults = append(faults, site)
		}
	}
	a.faults = faults
	delete(a.named, entry)
	page := os.Getpagesize()
	begin := -1
	for p := entry / page; p <= (entry+int(extent.length)-1)/page; p++ {
		a.pageLive[p]--
		if a.pageLive[p] == 0 {
			if begin < 0 {
				begin = p
			}
		} else if begin >= 0 {
			_ = pureDiscard(a.base+uintptr(begin*page), (p-begin)*page)
			begin = -1
		}
	}
	if begin >= 0 {
		end := (entry + int(extent.length) + page - 1) / page
		_ = pureDiscard(a.base+uintptr(begin*page), (end-begin)*page)
	}
	return int(extent.length)
}
