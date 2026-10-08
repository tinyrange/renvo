//go:build !renvo

package runimage

import (
	"fmt"
	"io"
	"strings"
	"unsafe"
)

// JITCodeWriter receives named, immutable code under arena serialization.
// It writes profiling data only; it must never reenter or execute the arena.
type JITCodeWriter interface {
	WriteCode(address uintptr, code []byte, name string) error
}

func (a *CodeArena) SetJITWriter(w JITCodeWriter) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.jitSymbols = w
}

// SetSymbolWriter enables Linux perf-map compatible records for subsequently
// named entries. The writer is trusted profiling output, never executable code.
// Installed offsets remain private; records are serialized with arena mutation.
func (a *CodeArena) SetSymbolWriter(w io.Writer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.symbols = w
}
func (a *CodeArena) NameEntry(entry int, name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.symbols == nil && a.jitSymbols == nil {
		return nil
	}
	if a.base == 0 || a.broken || entry < 0 || entry&15 != 0 || entry>>4 >= len(a.entries) || a.entries[entry>>4].Words == 0 || entry>>4 >= len(a.lengths) || len(name) == 0 || len(name) > 160 || strings.ContainsAny(name, "\r\n\x00") {
		return fmt.Errorf("invalid native symbol")
	}
	address, length := a.base+uintptr(entry), int(a.lengths[entry>>4])
	if length < 1 || length > a.used-entry {
		return fmt.Errorf("invalid native symbol extent")
	}
	if a.symbols != nil {
		if _, err := fmt.Fprintf(a.symbols, "%x %x %s\n", address, length, name); err != nil {
			return err
		}
	}
	if a.jitSymbols != nil {
		return a.jitSymbols.WriteCode(address, unsafe.Slice((*byte)(unsafe.Pointer(address)), length), name)
	}
	return nil
}
