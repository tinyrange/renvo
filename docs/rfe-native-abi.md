# RFE native-call ABI

Internal contract for trusted Go-hosted amd64/arm64 generated code, not the Go
ABI or a sandbox. Layout assertions in the runtime reject incompatible hosts.

`internal/rfeabi` owns the shared GC-visible page, descriptor and context types.
Emitter offsets compose in `backend/compiler_common_impl.go` and are checked
against those types before native admission. The foreign-call staging image
derives its indices from the same typed layout; C treats the context as opaque.

## Entries and ownership

An installed entry is a 16-byte-aligned arena offset, never a public executable
pointer. Standalone pure/memory entries establish a compatibility frame and call
an internal shared body. Shared bodies inherit the state/context and frame.
Loop bodies are dispatcher-only: standalone calls reject them. Legacy framed
leaves remain supported; dispatch restores the state base after their return.

| Role | amd64 | arm64 |
| --- | --- | --- |
| State argument / result scratch | AX | X0 |
| Context argument | DX | X1 |
| Shared state base | R11 | X11 |
| Frame base | RBP | X29 |
| Dispatcher state / context / descriptor | R12 / R13 / R14 | X19 / X20 / X21 |
| Static instruction count / remaining budget | R15 / RBX | X22 / X23 |

Leaf code preserves dispatcher-owned registers. Loops preserve the subset they
use on every return, including faults and side exits. Generated exit maps restore
architectural state; Go callbacks do not reconstruct guest registers.

## Frames and bounds

The dispatcher owns one 17,408-byte frame on a 64 KiB private stack. Shared
bodies do not grow that stack. Their frame displacements are biased by 128 bytes
to separate dispatcher saves/facts from SSA spills. FP-144 holds the context;
pure spills start at FP-152. Loop saves, phi temporaries, translation facts and
inherited output captures have separate slots. Emission checks the full peak
against the frame limit, including up to 2,048 IR records. Legacy nesting is
bounded; linked transitions return through the checked dispatcher.

Cold blocks contain at most 16 guest instructions; optimized regions at most
256. Legacy calls retire at most 64 instructions. Linux/amd64+cgo foreign
sessions independently permit at most 65,536; no-cgo/other-host and alias-safe
compatibility paths retain short calls. Host scheduling may aggregate separate
short calls but never replenish a call's budget inside native code.

## Admission and invalidation

Each aligned arena offset has private metadata: state/family, immutable admission
key, internal body offset, installed static count and admitted descriptor count.
Body offsets are checked against the installed image. Public descriptors cannot
supply internal executable addresses or grant admission.

Descriptor and prepared-proof tables each have 1,024 entries in four ways, with
full PC tags. Each selection checks ownership, state/count shape and budget.
Session admission epochs permit reuse only within the current serialized call;
wrap clears cached epochs. Appending code refreshes borrowed metadata views.
Close invalidates owner-bound prepared handles under the arena lock.

Memory accesses check canonical address, width, page tag and permissions.
Read and write facts are distinct. Facts may cross leaf transitions only within
one serialized call and reset at the next host boundary. Exact invariant-address
facts reuse translation, never loaded data. Generic stores check generation
overflow before changing the clock, page epoch or guest bytes. Code-versioned
loops may omit ordinary data-store generation updates only for an explicitly
registered memory context. All linked host boundaries check that registration;
a generic context cannot inherit this specialization. Executable stores take
the architectural slow path. Mapping/protection transitions establish fresh code
generations and invalidate engine links before admitting executable bytes.

## Foreign sessions and exits

Supported sessions stage state/context in pointer-free buffers, hold the arena
lock and pin every native-reachable allocation: state/context, pages, epochs,
clock, descriptors, proofs, metadata and private stack. Encoded borrowed addresses
are cleared before unpinning. Physical aliases use compatibility calls instead
of staging. The cgo bridge enters generated code on the private stack and returns
through the supported foreign-call boundary.

Misses, faults, traps, cold compilation and mapping work return to Go. Exact
retired prefixes, remaining budget and memory counts are validated before host
accounting. A slow-path instruction is not replayed after successful retirement.

## Optional fault-assisted memory

Linux/amd64 with cgo can opt into a reserved guest window through DirectGuest,
DirectHost and DirectSize context fields. Admission checks the fixed window
size; emitted code checks the full unsigned guest offset, never masks an address
into RAM. Only naturally aligned scalar stores use the direct path; other
accesses retain checked fallbacks. Holes are inaccessible, and executable guest
pages are not directly writable. Mapping ownership is serialized with native
calls; engines must stop before guest mappings close.

The arena validates, sorts and pins exact access-PC/recovery-PC/width records.
A thread-local C scope admits recovery only at those sites and within the scalar
access's effective address range. Other faults chain to the previous signal
handler. No Go callback runs in the handler. The direct fields are staged and
cleared with other borrowed addresses. This experimental mode is off by default.
