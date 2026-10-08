//go:build !renvo

package runimage

import (
	"fmt"
	"io"
	"runtime"
	"unsafe"
)

// WriteCodeSnapshot copies installed native bytes as profiling data while the
// owner is serialized. It does not expose an executable pointer, allocate more
// executable storage, or change entry admission. A complete marker is written
// only after all binary bytes have been accepted by the trusted host writer.
func (a *CodeArena) WriteCodeSnapshot(code, metadata io.Writer) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if code == nil || metadata == nil || a.base == 0 || a.broken || a.used < 1 || a.used > a.size || a.used > 8<<20 {
		return fmt.Errorf("invalid native-code snapshot")
	}
	// A writer may retain its input. Give it ordinary copied bytes, never
	// a slice into executable storage which Close will subsequently unmap.
	bytes := append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(a.base)), a.used)...)
	written, err := code.Write(bytes)
	if err != nil {
		return err
	}
	if written != len(bytes) {
		return io.ErrShortWrite
	}
	_, err = fmt.Fprintf(metadata, "arena %x %x %s complete\n", a.base, a.used, runtime.GOARCH)
	runtime.KeepAlive(a)
	return err
}
