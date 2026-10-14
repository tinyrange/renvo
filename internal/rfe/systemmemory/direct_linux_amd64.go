//go:build !renvo && linux && amd64 && cgo

package systemmemory

import (
	"fmt"
	"renvo.dev/internal/rfenativebridge"
	"syscall"
	"unsafe"
)

type directPage struct {
	offset     uint64
	prot       int
	generation uint64
	valid      bool
}
type directWindow struct {
	guest, hits, generation uint64
	view                    []byte
	pages                   []directPage
	active                  bool
}
type mappedRAM struct {
	fd       int
	data     []byte
	windows  map[uint64]*directWindow
	selected *directWindow
	closed   bool
}

func newDirectRAM(size int) ([]byte, directMemory, error) {
	if syscall.Getpagesize() != 4096 || !rfenativebridge.EnableFaults() {
		return make([]byte, size), nil, nil
	}
	fd, err := rfenativebridge.MemoryFD()
	if err != nil {
		return nil, nil, err
	}
	if err = syscall.Ftruncate(fd, int64(size)); err != nil {
		_ = syscall.Close(fd)
		return nil, nil, err
	}
	data, err := syscall.Mmap(fd, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		_ = syscall.Close(fd)
		return nil, nil, err
	}
	d := &mappedRAM{fd: fd, data: data, windows: make(map[uint64]*directWindow)}
	return data, d, nil
}
func (d *mappedRAM) create(guest uint64) *directWindow {
	if d.closed {
		return nil
	}
	if len(d.windows) >= 8 {
		// Reuse only a completely revoked reservation, never an active view.
		// forget cleared the context fields and set every page PROT_NONE;
		// refills occur outside native execution. No old page is reauthorized.
		for old, w := range d.windows {
			if w.active {
				continue
			}
			delete(d.windows, old)
			if d.selected == w {
				d.selected = nil
			}
			if len(w.view) == 0 {
				break
			}
			w.guest, w.hits, w.generation = guest, 0, 1
			clear(w.pages)
			d.windows[guest] = w
			return w
		}
		if len(d.windows) >= 8 {
			return nil
		}
	}
	view, err := syscall.Mmap(-1, 0, int(directWindowSize), syscall.PROT_NONE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return nil
	}
	w := &directWindow{guest: guest, view: view, generation: 1, pages: make([]directPage, directWindowSize/4096)}
	d.windows[guest] = w
	return w
}
func (d *mappedRAM) disable(w *directWindow) {
	// No native execution is active. Unmapping prevents any stale authority if
	// a protection operation fails. Native fields are cleared by the RAM owner.
	if len(w.view) != 0 {
		_ = syscall.Munmap(w.view)
	}
	w.view = nil
	w.active = false
}
func (d *mappedRAM) refill(address, offset, permissions uint64) {
	if d.closed || address >= 1<<48 || offset&4095 != address&4095 || offset >= uint64(len(d.data)) {
		return
	}
	physical := offset &^ uint64(4095)
	if uint64(len(d.data))-physical < 4096 {
		return
	}
	guest := address &^ (directWindowSize - 1)
	w := d.windows[guest]
	if w == nil {
		w = d.create(guest)
	}
	if w == nil || len(w.view) == 0 {
		return
	}
	at := (address - guest) &^ uint64(4095)
	page := &w.pages[at/4096]
	prot := syscall.PROT_NONE
	if permissions&1 != 0 {
		prot = syscall.PROT_READ
		if permissions&6 == 2 {
			prot |= syscall.PROT_WRITE
		}
	}
	if !page.valid || page.offset != physical {
		// MAP_FIXED is permitted ONLY for one aligned page wholly inside this
		// still-owned reservation. Neither guest VA nor a masked host address
		// is used as a host destination. Backing offsets were extent checked.
		target := uintptr(unsafe.Pointer(&w.view[at]))
		mapped, _, errno := syscall.Syscall6(syscall.SYS_MMAP, target, 4096, uintptr(prot), syscall.MAP_SHARED|syscall.MAP_FIXED, uintptr(d.fd), uintptr(physical))
		if errno != 0 || mapped != target {
			d.disable(w)
			return
		}
		*page = directPage{offset: physical, prot: prot, generation: w.generation, valid: true}
	} else if page.generation != w.generation || page.prot != prot {
		if syscall.Mprotect(w.view[at:at+4096], prot) != nil {
			d.disable(w)
			return
		}
		page.generation, page.prot = w.generation, prot
	}
	w.active = true
	if w.hits != ^uint64(0) {
		w.hits++
	}
	if d.selected == nil || len(d.selected.view) == 0 || w.hits > d.selected.hits {
		d.selected = w
	}
}
func (d *mappedRAM) forgetPage(address uint64) {
	w := d.windows[address&^(directWindowSize-1)]
	if w == nil || len(w.view) == 0 {
		return
	}
	at := (address - w.guest) &^ uint64(4095)
	p := &w.pages[at/4096]
	if p.valid && p.generation == w.generation {
		if syscall.Mprotect(w.view[at:at+4096], syscall.PROT_NONE) != nil {
			d.disable(w)
			return
		}
		p.generation = 0
	}
}
func (d *mappedRAM) forget() {
	// Heat and selection belong to the current mapping root, not cold boot.
	// Otherwise an inactive old-root winner can hide every new active view.
	d.selected = nil
	for _, w := range d.windows {
		w.hits = 0
		if !w.active || len(w.view) == 0 {
			continue
		}
		if syscall.Mprotect(w.view, syscall.PROT_NONE) != nil {
			d.disable(w)
			continue
		}
		w.active = false
		w.generation++
		if w.generation == 0 {
			d.disable(w)
		}
	}
}
func (d *mappedRAM) window() (uint64, uint64, uint64) {
	w := d.selected
	if d.closed || w == nil || len(w.view) == 0 || !w.active {
		return 0, 0, 0
	}
	return w.guest, uint64(uintptr(unsafe.Pointer(&w.view[0]))), directWindowSize
}
func (d *mappedRAM) close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	var first error
	for _, w := range d.windows {
		if len(w.view) != 0 {
			if err := syscall.Munmap(w.view); err != nil && first == nil {
				first = fmt.Errorf("native RAM view unmap: %w", err)
			}
			w.view = nil
		}
	}
	if err := syscall.Munmap(d.data); err != nil && first == nil {
		first = err
	}
	if err := syscall.Close(d.fd); err != nil && first == nil {
		first = err
	}
	d.data, d.windows, d.selected = nil, nil, nil
	return first
}
