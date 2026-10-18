# Opt-in desktop managed heap

All 33 Linux targets in `tools/qemu-smoke/targets.json` now have RBE/RTG adapters
for the growable, nonmoving managed heap; see `QEMU.md` for executed coverage.
Existing targets retain their arena allocator unless their definition opts in.
This is an initial synchronous collector, not a claim of production GC throughput
or pause-time guarantees.

## Use

From the repository root, with the hosted Renvo driver:

```sh
renvo -backend backend/definitions/linux_amd64_desktop.rbe \
  -t linux/amd64-desktop -heap-limit 64MiB -o app ./cmd/app
RENVO_HEAP_LIMIT=128MiB ./app
```

The supplied `.rbe` imports the canonical Linux definition, which declares a
separate `linux/amd64-desktop` target with `memory_strategy = desktop`. The same
target is built into the bundled compiler; `-backend` can be omitted. Omitting the field,
or setting `memory_strategy = arena`, preserves the existing strategy. This does
not automatically change every target based on the amount of RAM detected.
The resolver derives the internal `renvo_managed` build tag from the strategy;
custom definitions do not need to copy it, and switching back to arena removes
it. External definitions are validated using their resolved policy, not a
similarly named built-in target.

Arena-based `-system` profiles are not managed-heap configurations. They are
rejected when their effective target is desktop, including targets selected
inside the profile. Use `-heap-limit` directly for desktop builds.

Use `renvo.dev/x/runtime.Collect()` for an explicit synchronous collection:

```go
import "renvo.dev/x/runtime"

func afterBatch() {
    runtime.Collect()
}
```

The call is a no-op for arena-only targets (and when built with host Go).
Ordinary allocations do not require explicit frees or user-maintained root lists.
All normal allocation lowering, including strings, slices, maps, escaped values,
interfaces and closures, uses the managed allocator in desktop mode.

## Budget and failure behavior

`-heap-limit` sets the compiled managed-heap budget, not a reservation. Without it
there is no user cap, subject to native address-space and OS allocation limits.
Only a desktop executable reads `RENVO_HEAP_LIMIT`, once before global
initialization. A present override replaces the compiled limit and may raise or
lower it. It is deployment policy, not an immutable security boundary.

Both accept positive decimal bytes, optionally suffixed with `KiB`, `MiB`, `GiB`
or `TiB`. Zero, empty values, signs, whitespace, ambiguous units and overflow are
errors. The runtime supports limits through the positive signed native-word
range (31 value bits on 32-bit targets, 63 on 64-bit targets); the compiling host
must also represent the requested limit. A malformed
environment value produces a startup diagnostic and exits with status 2.

The budget includes whole mapped segments: live payloads, block/segment headers,
alignment, and retained free capacity, plus 160 bytes of fixed manager and
allocator bookkeeping. Mappings are acquired lazily in page-sized units, so a
budget smaller than a page plus metadata cannot service an allocation. A budget
smaller than the fixed bookkeeping fails at startup. Code, native stacks,
application globals, OS bookkeeping and other unmanaged memory are **not**
charged; this is not a process RSS limit.

Allocation searches reusable blocks, then collects and retries, then grows within
the remaining budget. Preferred segments start at 64 KiB and grow geometrically;
a failed preferred OS mapping retries at the minimum size. After collection and
retry cannot satisfy a request, allocation follows the normal OOM panic path.
A program can recover from allocation OOM using ordinary `defer`/`recover`.
There is no application recovery handler during initial process setup.

## Collector and supported execution model

The native collector is defined in `backend/definitions/managed_heap64.rtg`;
It provides both `managedHeap32Source()` and `managedHeap64Source()`. The
`managed_linux*.rtg` adapters supply Linux mappings and boundary emitters for
maintained targets. Experimental RBEs embed the shared collector so each remains
standalone; `TestManagedEmbeddedCollectorConsistency` rejects drift from the
canonical source. Synchronize those embedded copies when changing the collector.
Both prepared and bundled compilers project these same definition hooks. It uses
a free-block allocation cursor with first-fit fallback, nonrecursive conservative
tracing with an intrusive mark worklist, sweeping,
adjacent-block coalescing and release of completely empty segments. Reused
payloads are zeroed. Collector bookkeeping requires no heap allocations.

At each collection boundary, the adapter saves all general-purpose registers and
scans the active native stack and global storage. It excludes the manager's own
segment-list state from roots. Payloads are scanned as native words, including
interior pointers and cycles. Ordinary panic/defer, interface, closure and map
storage is retained through those roots. Runtime helpers do not consume the
application's deferred-recovery permission.

Conservative values can retain unreachable storage. Encoded or deliberately
unaligned pointers hidden in raw bytes are not supported roots. The initial
implementation supports one serialized native stack, not concurrent mutators or
externally switched/suspended stacks. The prepared stack API is unsupported;
managed configurations reject stack-switch and JIT intrinsics. Arena marks
return zero, resets/discards are no-ops, and persistence preserves identity:
these are lifetime hints, never explicit frees on the managed heap. Native FFI/C, object,
kernel-module and linked-image outputs are not supported in this first target.
Desktop definitions without a complete, typed managed ABI contract are rejected.
There is no OS/ISA/ABI-name allowlist for managed allocation.

## Self-host status

The compiler self-hosts on Linux/amd64 with the desktop heap. Select
`-t linux/amd64-desktop -heap-limit 256MiB -s` when compiling `./cmd/renvo`, and
use the same options with each resulting compiler. The managed build uses its
bundled in-process backend on the native stack and disables script JIT setup.
Arbitrary external `.rbe` resolution is still a hosted-driver feature.

`TestManagedHeapSelfHost` builds stages 1, 2 and 3, requires byte-identical stage-2
and stage-3 binaries, verifies the environment heap limit in every stage, and
runs pressure/explicit-collection programs compiled by stages 1 and 3. The test
keeps the existing 256 MiB compiler budget and 60-second per-process deadline.
See `STRESS.md` for coverage and limits; this is not a performance-gate result.

## Core model and validation

The Go package in this directory remains an independently testable allocator
core with pluggable `Memory` and mandatory `Roots` providers. The definition-owned
runtime is a backend-subset adaptation, not an import of that Go package. Core
provider tests alone do not validate automatic application roots; native tests
in `frontend_tests/frontend_managed_heap_test.go` exercise the integrated path.

The provider must return stable, aligned, zeroed storage of the requested size,
retain it independently of any host collector and release it on request. It must
not allocate from the heap it implements. Charge fixed adapter state through
`Config.Overhead`; variable metadata belongs inside charged segments. The core's
`Alloc` returns nil and `ResolveLimit` only parses/resolves values. The native
adapter owns startup environment reads and conversion to runtime panics.

Core tests use an independent graph reachability oracle and cover cycles,
interior pointers, reuse/release, failed root enumeration, hard limits, host
failure, overflow, reentrancy and allocation-free collection. The smoke program
under `testdata/smoke` verifies the core compiled by Renvo with explicit roots.
Native acceptance tests additionally cover automatic pressure collection,
explicit collection, initialized globals, containers, zeroing, growth, compile
and environment budgets, invalid overrides and OOM recovery.

The native runtime uses an in-segment page-to-block index for conservative
pointer lookup and block-header links for its mark worklist. Index storage is
charged to the heap budget; tracing itself allocates no memory. The independent
Go core still uses linear lookup and fixed-point tracing. Neither implementation
promises bounded pauses or elimination of conservative retention/fragmentation.

## Adding a platform through RBE/RTG

The common compiler’s managed-heap path contains no Linux syscall numbers,
x86 register-save bytes, or managed target-name whitelist. An external `.rbe` can import or author these
runtime bindings without changing compiler Go code:

```text
managed_heap_abi = 1
managed_word_bits = 64
managed_source = go adapter.runtimeSource
managed_startup = go adapter.startup
managed_boundary = go adapter.boundary
```

The containing target opts in with `memory_strategy = desktop`. Hook signatures:

- `runtimeSource() string`: return backend-subset source defining
  `__renvoManagedInit`, `__renvoManagedRequest`, and
  `__renvoManagedCollectBoundary`. Those names are the compiler/runtime ABI.
- `startup(out *RTGEmitter, limit int, label int)`: emit initialization before
  application global initializers. `label` identifies `__renvoManagedInit`.
  Zero limit means uncapped; the adapter supplies stack and global-root bounds.
- `boundary(out *RTGEmitter, label int)`: capture all pointer-bearing registers
  and roots, pass the request from the primary register to the runtime function,
  and return its result in primary while restoring application state. Allocation
  failure returns zero for the compiler's existing OOM panic path.

The supplied 32/64-bit collector expects Init(limit, stack, globalsBegin,
globalsEnd), Request(size, roots), and CollectBoundary(ignored, roots), using
native-word arguments/results. It can be shared by adapters supplying
`__renvoManagedMap`, `__renvoManagedRelease`, `__renvoManagedFatal`,
`__renvoManagedEnvironment`, `__renvoManagedRoots`, and `__renvoManagedPageSize`.
See `managed_linux_amd64.rtg` for their concrete signatures. Maps must be stable,
zeroed, word-aligned storage; map failure returns zero. Release reports success;
fatal must not return; environment returns a NUL-terminated override or zero.
Page size is a power-of-two mapping granularity (at least 64 bytes). The adapter
must enumerate all supported live stack/register roots and must not allocate
through the heap it implements. Globals and manager-state exclusion are handled
by the shared collector. Extra adapter metadata must also be budgeted.

This is **definition-authored platform code, not a magic configuration switch**.
A genuinely new OS still needs its mapping/environment/startup implementation;
a new calling convention needs correct root capture. The reusable collector
supports native 32- and 64-bit words; narrower or mixed-width layouts need a
compatible collector implementation.
The resolver checks ABI version, hook signatures and matching descriptor widths,
not the correctness of arbitrary adapter source. Ordinary architecture/emitter
support is still a prerequisite.

The complete Linux QEMU matrix passes managed pressure, explicit collection,
word-width-sensitive object graphs, interior/global roots, OOM recovery, and
compile/environment budget checks in both endian modes where available. Linux/amd64
also has native stress and three-stage managed compiler self-host coverage.
This is not native-hardware qualification or compiler self-host proof for the
other 32 targets. Other operating systems remain unqualified. Adding a *built-in*
target additionally requires the normal registry/generation step; hosted external
RBE use does not.

To opt in on an existing supplied Linux RBE, set `memory_strategy = desktop` in
its selected `target` declaration and pass `-backend path/to/your.rbe`,
`-t <target>` and `-heap-limit 64MiB`. Leave its runtime bindings intact. Maintained `.rtg` definitions
use relative imports; preserve those files together when copying. Experimental
RBEs remain single-file definitions. Ordinary targets still default to arena;
only `linux/amd64-desktop` is a separately named bundled desktop target.

HPPA retains its existing 4 MiB private native stack, outside the managed heap
budget. Its adapter captures Linux envp before switching stacks and excludes the
private stack reservation from global roots, scanning only its active portion.
