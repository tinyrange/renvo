//go:build !renvo && cgo && linux && amd64

#define _GNU_SOURCE
#include <stdint.h>
#include <stddef.h>
#include <errno.h>
#include <signal.h>
#include <pthread.h>
#include <ucontext.h>
#include <sys/mman.h>

extern void renvo_rfe_enter(uintptr_t, uint64_t *, void *, uintptr_t);
struct rfe_fault_scope { const uintptr_t *sites; size_t count; struct rfe_fault_scope *previous; };
// No dynamic TLS allocation or Go calls are permitted in the signal handler.
static __thread struct rfe_fault_scope *volatile rfe_scope __attribute__((tls_model("initial-exec")));
static pthread_once_t rfe_once = PTHREAD_ONCE_INIT;
static struct sigaction rfe_old_segv, rfe_old_bus;
static int rfe_setup_error;

static void rfe_fault(int sig, siginfo_t *info, void *raw) {
    int saved_errno = errno;
    ucontext_t *uc = (ucontext_t *)raw;
    struct rfe_fault_scope *scope = rfe_scope;
    if (scope && info->si_code > 0) {
        uintptr_t pc = (uintptr_t)uc->uc_mcontext.gregs[REG_RIP];
        size_t lo = 0, hi = scope->count;
        while (lo < hi) {
            size_t mid = lo + (hi-lo)/2;
            if (scope->sites[mid*3] < pc) lo = mid+1; else hi = mid;
        }
        if (lo < scope->count && scope->sites[lo*3] == pc) {
            uintptr_t address = (uintptr_t)info->si_addr;
            uintptr_t base = (uintptr_t)uc->uc_mcontext.gregs[REG_RSI];
            // Only the registered scalar MOV, using checked RSI as its full
            // effective address, is recoverable. Never swallow an unrelated
            // native/Go fault, or a signal merely delivered during a session.
            if (address >= base && address-base < scope->sites[lo*3+2]) {
                uc->uc_mcontext.gregs[REG_RIP] = scope->sites[lo*3+1];
                errno = saved_errno;
                return;
            }
        }
    }
    const struct sigaction *old = sig == SIGSEGV ? &rfe_old_segv : &rfe_old_bus;
    errno = saved_errno;
    if (old->sa_handler == SIG_DFL) {
        sigaction(sig, old, NULL);
        raise(sig);
    } else if (old->sa_handler != SIG_IGN) {
        if (old->sa_flags & SA_SIGINFO) old->sa_sigaction(sig, info, raw);
        else old->sa_handler(sig);
    }
}
static void rfe_setup(void) {
    struct sigaction action = {0};
    action.sa_sigaction = rfe_fault;
    action.sa_flags = SA_SIGINFO | SA_ONSTACK;
    sigemptyset(&action.sa_mask);
    sigaddset(&action.sa_mask, SIGSEGV);
    sigaddset(&action.sa_mask, SIGBUS);
    // Publish both previous actions before either handler can run on another
    // thread. Installing a second third-party handler concurrently is not
    // supported; the process must serialize signal-handler ownership changes.
    if (sigaction(SIGSEGV, NULL, &rfe_old_segv) != 0 ||
        sigaction(SIGBUS, NULL, &rfe_old_bus) != 0) { rfe_setup_error = errno; return; }
    if (sigaction(SIGSEGV, &action, NULL) != 0) { rfe_setup_error = errno; return; }
    if (sigaction(SIGBUS, &action, NULL) != 0) {
        rfe_setup_error = errno;
        sigaction(SIGSEGV, &rfe_old_segv, NULL);
    }
}
int renvo_rfe_enable_faults(void) {
    int err = pthread_once(&rfe_once, rfe_setup);
    return err ? err : rfe_setup_error;
}
void renvo_rfe_enter_faults(uintptr_t entry, uint64_t *state, void *context, uintptr_t top,
                            const uintptr_t *sites, size_t count) {
    struct rfe_fault_scope scope = { sites, count, rfe_scope };
    rfe_scope = &scope;
    renvo_rfe_enter(entry, state, context, top);
    rfe_scope = scope.previous;
}
int renvo_rfe_memory_fd(void) { return memfd_create("renvo-guest", MFD_CLOEXEC); }
