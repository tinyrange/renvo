//go:build !renvo && cgo && linux && amd64

#include <stdint.h>
#include <stddef.h>
#include <time.h>
extern void renvo_rfe_enter_faults(uintptr_t, uint64_t *, void *, uintptr_t,
                                  const uintptr_t *, size_t);
uint64_t renvo_rfe_monotonic_ns(void) {
    struct timespec ts;
    if (clock_gettime(CLOCK_MONOTONIC, &ts) != 0) return 0;
    return (uint64_t)ts.tv_sec * UINT64_C(1000000000) + (uint64_t)ts.tv_nsec;
}
// The caller supplies validated, pinned words in the pointer-free ABI image.
// Each invocation of generated code retains its original <=64 instruction
// bound. The aggregate foreign session retains its independent resource bound.
int renvo_rfe_enter_until(uintptr_t entry, uint64_t *state, void *context,
                         uintptr_t top, const uintptr_t *sites, size_t count,
                         uint64_t *remaining, uint64_t *total, uint64_t *memory,
                         uint64_t deadline) {
    uint64_t *prefix = (uint64_t *)context;
    uint64_t budget = *remaining;
    int expired = 0;
    while (*total < budget) {
        uint64_t now = renvo_rfe_monotonic_ns();
        if (!now || now >= deadline) { expired = 1; break; }
        uint64_t before = *total;
        uint64_t slice = budget - before;
        if (slice > 64) slice = 64;
        *remaining = slice;
        renvo_rfe_enter_faults(entry, state, context, top, sites, count);
        if (*total < before || *total-before > slice ||
            *remaining != slice-(*total-before) || *memory > *total ||
            (prefix[1] > 2 && prefix[1] != 4)) {
            prefix[1] = 3;
            break;
        }
        // Faults and guards are architectural boundaries. A successful side
        // exit can redispatch, but zero progress must return for host discovery.
        if (prefix[1] == 1 || prefix[1] == 2 || *total == before) break;
        if (prefix[1] == 4) prefix[1] = 0;
    }
    *remaining = *total <= budget ? budget-*total : 0;
    return expired;
}
