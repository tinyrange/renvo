//go:build !renvo && cgo && linux && amd64

// Package rfenativebridge owns the supported Go-to-C scheduling boundary for
// generated RFE code. Admission, bounded budgets, roots and pinning are owned
// by runimage; this package never obtains or caches an executable address.
package rfenativebridge

/*
#include <stdint.h>
void renvo_rfe_enter(uintptr_t entry, uint64_t *state, void *context, uintptr_t top);
*/
import "C"

import "unsafe"

const Available = true

// Call accepts dedicated pointer-free Go allocations containing the native
// ABI image. The caller roots and pins every encoded address until return.
func Call(entry uintptr, state, context unsafe.Pointer, top uintptr) {
	C.renvo_rfe_enter(C.uintptr_t(entry), (*C.uint64_t)(state), context, C.uintptr_t(top))
}
