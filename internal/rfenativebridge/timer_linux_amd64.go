//go:build !renvo && cgo && linux && amd64

package rfenativebridge

/*
#include <stdint.h>
#include <stddef.h>
uint64_t renvo_rfe_monotonic_ns(void);
int renvo_rfe_enter_until(uintptr_t, uint64_t *, void *, uintptr_t,
                        const uintptr_t *, size_t, uint64_t *, uint64_t *,
                        uint64_t *, uint64_t);
*/
import "C"
import "unsafe"

// MonotonicNS is CLOCK_MONOTONIC, shared by the hosted oscillator and native
// timer polling. A zero result disables deadline sessions conservatively.
func MonotonicNS() uint64 { return uint64(C.renvo_rfe_monotonic_ns()) }

func CallUntil(entry uintptr, state, context unsafe.Pointer, top uintptr, sites []FaultSite,
	remaining, total, memory *uint64, deadline uint64) bool {
	var faults *C.uintptr_t
	if len(sites) != 0 {
		faults = (*C.uintptr_t)(unsafe.Pointer(&sites[0]))
	}
	return C.renvo_rfe_enter_until(C.uintptr_t(entry), (*C.uint64_t)(state), context,
		C.uintptr_t(top), faults, C.size_t(len(sites)), (*C.uint64_t)(remaining),
		(*C.uint64_t)(total), (*C.uint64_t)(memory), C.uint64_t(deadline)) != 0
}
