//go:build !renvo && cgo && linux && amd64

// Package rfenativebridge owns the supported Go-to-C scheduling boundary for
// generated RFE code. Admission, bounded budgets, roots and pinning are owned
// by runimage; this package never obtains or caches an executable address.
package rfenativebridge

/*
#include <stdint.h>
#include <stddef.h>

typedef struct {
	uint64_t number;
	unsigned char *data;
	uint64_t permissions;
	uint64_t *epoch;
} renvo_rfe_page;
typedef struct {
	uint64_t pc, entry, instructions, reserved;
	unsigned char prefix[17], padding[15];
} renvo_rfe_descriptor;
typedef struct {
	uint64_t retired, status, address;
	uint64_t *clock;
	renvo_rfe_page pages[64];
	uint64_t remaining, total, memory_total, code_view[4];
	renvo_rfe_descriptor blocks[1024];
	uint64_t loop_exits, loop_iterations, prepared_targets, descriptor_base, admission_epoch;
} renvo_rfe_context;
_Static_assert(sizeof(renvo_rfe_context) == 67712, "RFE context size");
_Static_assert(offsetof(renvo_rfe_context, pages) == 32, "RFE pages offset");
_Static_assert(offsetof(renvo_rfe_context, remaining) == 2080, "RFE budget offset");
_Static_assert(offsetof(renvo_rfe_context, blocks) == 2136, "RFE descriptors offset");
_Static_assert(offsetof(renvo_rfe_context, prepared_targets) == 67688, "RFE proofs offset");
_Static_assert(offsetof(renvo_rfe_context, descriptor_base) == 67696, "RFE descriptor base offset");
_Static_assert(offsetof(renvo_rfe_context, admission_epoch) == 67704, "RFE admission epoch offset");
void renvo_rfe_enter(uintptr_t entry, uint64_t *state, renvo_rfe_context *context, uintptr_t top);
*/
import "C"

import "unsafe"

const Available = true

// Call accepts dedicated pointer-free Go allocations containing the native
// ABI image. The caller roots and pins every encoded address until return.
func Call(entry uintptr, state, context unsafe.Pointer, top uintptr) {
	C.renvo_rfe_enter(C.uintptr_t(entry), (*C.uint64_t)(state), (*C.renvo_rfe_context)(context), C.uintptr_t(top))
}
