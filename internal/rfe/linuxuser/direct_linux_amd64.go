//go:build !renvo && linux && amd64 && cgo

package linuxuser

import (
	"fmt"
	"renvo.dev/internal/rfenativebridge"
	"runtime"
	"syscall"
	"unsafe"
)

// Each window has a checked host-service alias and a protection-enforced JIT
// view of the same sparse anonymous file. Neither view is ever executable.
// No MAP_FIXED request, address masking, or guest-selected host address occurs.
type directWindow struct {
	// Only pages actually allocated in this window may gain JIT permissions.
	// A previous allocation may have fallen back before this window existed.
	backed      [DirectWindowSize / PageSize]bool
	hits        uint64
	guest       uint64
	alias, view []byte
	disabled    bool
}
type mappedMemory struct {
	selected *directWindow
	windows  map[uint64]*directWindow
	closed   bool
}

func newDirectMemory() (directMemory, error) {
	if syscall.Getpagesize() != int(PageSize) || !rfenativebridge.EnableFaults() {
		return nil, directUnavailable()
	}
	d := &mappedMemory{windows: make(map[uint64]*directWindow)}
	runtime.SetFinalizer(d, func(d *mappedMemory) { _ = d.close() })
	return d, nil
}
func (d *mappedMemory) create(guest uint64) *directWindow {
	// Bound VA reservations independently of physical backing. Widely scattered
	// mappings beyond this limit remain ordinary checked pages.
	if d.closed || len(d.windows) >= 8 {
		return nil
	}
	fd, err := rfenativebridge.MemoryFD()
	if err != nil {
		return nil
	}
	defer syscall.Close(fd)
	if syscall.Ftruncate(fd, int64(DirectWindowSize)) != nil {
		return nil
	}
	alias, err := syscall.Mmap(fd, 0, int(DirectWindowSize), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil
	}
	view, err := syscall.Mmap(fd, 0, int(DirectWindowSize), syscall.PROT_NONE, syscall.MAP_SHARED)
	if err != nil {
		_ = syscall.Munmap(alias)
		return nil
	}
	w := &directWindow{guest: guest, alias: alias, view: view}
	d.windows[guest] = w
	return w
}
func directProtection(permissions uint8) int {
	// x86 cannot enforce write-only memory. Such pages use the checked path.
	// Writes to executable guest pages must exit for code invalidation.
	if permissions&ReadPermission == 0 {
		return syscall.PROT_NONE
	}
	prot := syscall.PROT_READ
	if permissions&(WritePermission|ExecutePermission) == WritePermission {
		prot |= syscall.PROT_WRITE
	}
	return prot
}
func (d *mappedMemory) page(address uint64, permissions uint8) *[4096]byte {
	guest := address &^ (DirectWindowSize - 1)
	w := d.windows[guest]
	if w == nil {
		w = d.create(guest)
	}
	if w == nil {
		return nil
	}
	at := address - guest
	data := (*[4096]byte)(unsafe.Pointer(&w.alias[at]))
	*data = [4096]byte{} // remapping must never expose the old page's contents
	w.backed[at/PageSize] = true
	if !w.disabled && syscall.Mprotect(w.view[at:at+PageSize], directProtection(permissions)) != nil {
		w.disabled = true
	}
	return data
}
func (d *mappedMemory) protect(address, length uint64, permissions uint8) {
	for at := address; at < address+length; at += PageSize {
		w := d.windows[at&^(DirectWindowSize-1)]
		if w == nil || w.disabled {
			continue
		}
		offset := at - w.guest
		if !w.backed[offset/PageSize] {
			continue
		}
		if syscall.Mprotect(w.view[offset:offset+PageSize], directProtection(permissions)) != nil {
			w.disabled = true
		}
	}
}
func (d *mappedMemory) unmap(address, length uint64) {
	d.protect(address, length, 0)
	// The reservation remains owned, with PROT_NONE holes, until Close.
	// Drop backing of unmapped pages without changing either view's address.
	for at := address; at < address+length; at += PageSize {
		w := d.windows[at&^(DirectWindowSize-1)]
		if w == nil {
			continue
		}
		offset := at - w.guest
		w.backed[offset/PageSize] = false
		// MADV_REMOVE punches the shared shmem backing, unlike MADV_DONTNEED.
		_ = syscall.Madvise(w.alias[offset:offset+PageSize], syscall.MADV_REMOVE)
	}
}
func (d *mappedMemory) window(address uint64) (uint64, uint64, uint64) {
	if d.closed {
		return 0, 0, 0
	}
	w := d.windows[address&^(DirectWindowSize-1)]
	if w != nil && !w.disabled {
		if w.hits != ^uint64(0) {
			w.hits++
		}
		if d.selected == nil || d.selected.disabled || w.hits > d.selected.hits {
			d.selected = w
		}
	}
	w = d.selected
	if w == nil || w.disabled {
		return 0, 0, 0
	}
	return w.guest, uint64(uintptr(unsafe.Pointer(&w.view[0]))), DirectWindowSize
}
func (d *mappedMemory) close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	var first error
	for _, w := range d.windows {
		if err := syscall.Munmap(w.view); err != nil && first == nil {
			first = fmt.Errorf("guest view unmap: %w", err)
		}
		if err := syscall.Munmap(w.alias); err != nil && first == nil {
			first = fmt.Errorf("guest alias unmap: %w", err)
		}
		w.disabled = true
	}
	d.windows = nil
	return first
}
