//go:build !renvo && cgo && linux && amd64

#include <stdint.h>
#include <time.h>
static inline uint64_t renvo_rfe_read_monotonic_ns(void) {
    struct timespec ts;
    if (clock_gettime(CLOCK_MONOTONIC, &ts) != 0) return 0;
    return (uint64_t)ts.tv_sec * UINT64_C(1000000000) + (uint64_t)ts.tv_nsec;
}
uint64_t renvo_rfe_monotonic_ns(void) {
    return renvo_rfe_read_monotonic_ns();
}
// Called on the owned foreign stack by the admitted dispatcher. No Go callback
// or pointer graph mutation is possible; a clock failure expires conservatively.
static uint64_t renvo_rfe_timer_due(uint64_t deadline) {
    uint64_t now = renvo_rfe_read_monotonic_ns();
    return !now || now >= deadline;
}
uintptr_t renvo_rfe_timer_function(void) {
    return (uintptr_t)&renvo_rfe_timer_due;
}
