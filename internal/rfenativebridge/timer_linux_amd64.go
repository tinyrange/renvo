//go:build !renvo && cgo && linux && amd64

package rfenativebridge

/*
#include <stdint.h>
uint64_t renvo_rfe_monotonic_ns(void);
uintptr_t renvo_rfe_timer_function(void);
*/
import "C"

// MonotonicNS is CLOCK_MONOTONIC, shared by the hosted oscillator and native
// timer polling. A zero result disables deadline sessions conservatively.
func MonotonicNS() uint64 { return uint64(C.renvo_rfe_monotonic_ns()) }

// TimerFunction is the fixed, host-owned SysV poll. It takes a monotonic deadline
// and returns one when due (including clock failure). Only non-aliasing pinned
// staging images may borrow it; public context words are never authority.
func TimerFunction() uint64 { return uint64(C.renvo_rfe_timer_function()) }
