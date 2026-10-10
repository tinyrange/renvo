//go:build !renvo && cgo && linux && amd64

// Package rfenativebridge owns the supported Go-to-C scheduling boundary for
// generated RFE code. Admission, bounded budgets, roots and pinning are owned
// by runimage; this package never obtains or caches an executable address.
package rfenativebridge

/*
#include <stdint.h>
#include <stddef.h>
void renvo_rfe_enter(uintptr_t entry, uint64_t *state, void *context, uintptr_t top);
int renvo_rfe_enable_faults(void);
void renvo_rfe_enter_faults(uintptr_t, uint64_t *, void *, uintptr_t, const uintptr_t *, size_t);
int renvo_rfe_memory_fd(void);
*/
import "C"

import "unsafe"

const Available = true

// Call accepts dedicated pointer-free Go allocations containing the native
// ABI image. The caller roots and pins every encoded address until return.
func Call(entry uintptr, state, context unsafe.Pointer, top uintptr) {
	C.renvo_rfe_enter(C.uintptr_t(entry), (*C.uint64_t)(state), context, C.uintptr_t(top))
}

func EnableFaults() bool { return C.renvo_rfe_enable_faults() == 0 }
func MemoryFD() (int, error) {
	fd, err := C.renvo_rfe_memory_fd()
	if fd < 0 {
		return -1, err
	}
	return int(fd), nil
}

// CallFaults borrows an immutable, pinned, pointer-free table only for this
// foreign call. The signal path redirects within generated code; no longjmp
// crosses Go frames, and architectural restoration runs outside the handler.
func CallFaults(entry uintptr, state, context unsafe.Pointer, top uintptr, sites []FaultSite) {
	if len(sites) == 0 {
		Call(entry, state, context, top)
		return
	}
	C.renvo_rfe_enter_faults(C.uintptr_t(entry), (*C.uint64_t)(state), context, C.uintptr_t(top), (*C.uintptr_t)(unsafe.Pointer(&sites[0])), C.size_t(len(sites)))
}
