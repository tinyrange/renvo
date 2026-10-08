# RFE native-call ABI: shared frames (v2)

> Current benchmark summary: `aarch64-coremark-score.md`. Dated sections below
> retain development history; older results and proposals are not current claims.
> Historical raw trial/profile filenames refer to local artifacts not included
> in this PR. Current score evidence is checked in alongside the summary.

This is the internal Go-hosted amd64/arm64 RFE execution ABI, not the Go ABI,
a guest ISA extension, or a general-purpose native-code sandbox. Generated code
and host contexts are trusted. Current execution evidence is Linux/amd64 only;
arm64 selection exists but has not been executed on arm64 hardware here.

## Entry kinds

- **Standalone pure/memory entry:** an installed, 16-byte-aligned arena offset.
  It receives the architectural state pointer in AX/X0 and, for memory, the
  trusted MemoryContext pointer in DX/X1. It establishes a compatibility frame
  and calls the shared body. Public Call/CallContext enter only this start.
- **Shared pure/memory body:** an internal relative offset in that same image.
  It inherits the owner's frame, R11/X11 state base and context slot. It does
  not create/destroy a frame or make calls. Only native dispatch uses it.
- **Shared loop entry:** a dispatcher-only body at the installed start. It
  additionally receives the remaining instruction budget in RBX/X23. It keeps
  loop SSA values in registers/private spills and uses native precise exit maps.
  Public standalone calls reject the loop family.
- **Legacy framed entry:** remains supported by native linking. A legacy leaf
  may use R11/X11 as scratch; dispatch restores the state base after every call.

The old raw pure/memory/loop emitters remain available. Runtime compilation
uses RenvoEmitSharedBlock and RenvoEmitSharedLoopBlock. No archive source API
change is required: Native.Compile, CompileMemory, CompileLoop and their call
interfaces retain their signatures.

## Register ownership

| Role | amd64 | arm64 |
| --- | --- | --- |
| Stable architectural state base in shared bodies | R11 | X11 |
| Frame owner / base | RBP | X29 |
| Dispatcher state pointer | R12 | X19 |
| Dispatcher context pointer | R13 | X20 |
| Dispatcher descriptor | R14 | X21 |
| Dispatcher static body count | R15 | X22 |
| Remaining quantum budget | RBX | X23 |
| Ordinary argument / expression scratch | AX, DX, CX | X0, X1, X2 |

Shared bodies preserve the frame and dispatcher-owned registers. Ordinary SSA
uses the documented scratch pools. Loop allocation may use dispatcher-owned
registers, but preserves only those actually used, plus its budget/iteration
registers. Proven acyclic traces have no loop counters and can use those
registers for SSA instead. All paths, including faults and branch side exits, share that restore.
The dispatcher saves/restores its caller's owned registers once per quantum.
Inherited-output captures and stack-destination phi moves store directly from
an allocated SSA register when possible, without an extra scratch move. A
tentative generation-clock increment is checked for unsigned wrap to zero
before either the clock, epoch, or guest bytes are changed.
There is no guest-state assignment/reconstruction callback to Go.

## Frame ownership and partition

The dispatcher reserves **17,408 bytes** once on the existing **64 KiB private
stack**. Shared bodies borrow it without changing SP or FP:

| FP-relative area | Ownership |
| --- | --- |
| First 128 bytes below FP | Dispatcher-private saved registers, entry metadata and call-local read/write translation facts |
| FP-144 | Persistent trusted MemoryContext pointer for this quantum |
| Below the 128-byte prefix | Block-private SSA spills, loop preservation, phi temporary and invariant-access facts |

Every shared RFE frame displacement is biased by 128 bytes. Pure blocks also
reserve the context slot: their first SSA spill is FP-152, never FP-144. Memory
blocks use the same slot convention. Loop scratch starts farther down with its
own nonoverlapping save, phi and translation areas. Context/state bases are
established by the dispatcher or standalone compatibility owner, not each body.

The emitter checks each shared body's peak against the frame size. The bound
covers 2048 records all spilled plus 16 invariant-access facts:
`128 + 160 + 16*24 + 2048*8 = 17,056` bytes before inherited-output
captures. Such captures consume additional private spill slots; the full peak,
including them, is checked against 17,408 bytes and larger entries are rejected. Legacy framed code may nest once
below the dispatcher frame; the repository emitters' maximum frames still fit
within the unchanged private-stack allocation. Shared bodies cannot recurse or make nested
body calls. The opt-in checked continuation described below may tail-jump to
another independently admitted shared-loop body without growing the stack.

Shared scratch is reusable, not implicitly zeroed. Each SSA value is defined
before use. Exact invariant-access facts reset on each loop entry. Shared
read/write page facts may cross generated leaf transitions only within the
same serialized native call; the dispatcher and standalone compatibility owner
initialize them on every call. No fact survives a Go call boundary or a mapping
operation. Full 52-bit page tags, separate read/write permissions and width
checks still guard every variable-address access; every store allocates a fresh
generation before changing data.

## Checked dual entries and metadata

One private sixteen-byte record exists per 16-byte arena offset:

| Byte offset | Field |
| --- | --- |
| 0 | uint16 installed state/family metadata |
| 2 | uint16 immutable link-admission key |
| 4 | uint32 relative internal-body offset; zero selects the installed start |
| 8 | uint32 exact installed static loop count |
| 12 | uint32 exact immutable cold-admitted descriptor count |

The arena validates this layout at construction. Body offsets are checked
against the installed image length and committed under the same lock as code
installation. The native dispatcher validates the **public installed offset**
and exact state/count shape before adding the private body offset. Public link
records cannot supply or modify it. An internal body is not separately entered
in the public installed-entry table, even if its address happens to be aligned.
Appending code preserves records and refreshes the borrowed metadata pointer.

The table's logical payload is bounded by code capacity: at the default 8 MiB
code ceiling it is at most 8 MiB, excluding Go slice-capacity
overhead. The existing exact-length table is unchanged. There is no larger code
arena, private stack or guest RAM allocation hidden in the ABI change.

## Progress, guards and remaining limits

PC/full-tag checks, exact budgets, entry family/state/count checks, code-context
invalidation, memory permission checks and precise fault retirement remain.
Guest architecture is complete at an observable exit. Branch-side exits may
continue native dispatch; access/trap slow exits return before the pending
instruction without replaying the completed prefix.

The cold-block ceiling remains **16 guest instructions**, independent of the
optimized-region ceiling, now **256**, and supported foreign-session budget,
now **65,536 dynamic guest instructions**. Legacy assembly calls remain capped
at **64**. See the session contract below. Native linking still returns to Go
for misses, faults, traps, cold admission and bounded session scheduling.

The default path fixes frame ownership and repeated pointer/context setup,
while materializing values between independent blocks. A later opt-in AMD64
convention transfers one value between memory-free regions, described below.
General cross-block allocation, memory-region residency and return-progress
registers remain separate follow-up work. Standalone compatibility calls intentionally retain
frame setup and gain one internal CALL/RET; the optimization targets linked Run.


## Prepared dispatcher calls

`PrepareLinkedCall` validates an installed dispatcher once and returns an opaque
owner-bound handle with immutable entry, state size and private-stack top.
`LinkedCall.Call` still locks the original arena, rejects closed/broken owners
and wrong state/context/view arguments, borrows the current append-refreshed arena
view and clears it before unlocking. The handle is not an executable-address
escape and cannot authorize a leaf target; all native descriptor/admission guards
remain. The typed arena slices continue to root the stack and metadata even
though the private handle/view carry integer addresses. Close invalidates the
handle by closing its owner; no handle or view can revive that owner.

Register progress/status returns were investigated in batch 10 but not retained:
the selected implementation still uses the checked context progress convention.

## Bounded host scheduling and loop selection (batch 11)

`LinkedCall.CallBatch` holds the same arena lock/view for at most sixteen
**separate** dispatcher calls. Every call switches to the private stack and
returns to Go; each receives a fresh budget of at most 64. `Engine.Run` uses this legacy path only with conservative session settings.
The Linux process uses `Engine.RunQuanta`, which returns early on a
miss, fault, zero progress, exhausted ceiling or publication requiring cold
preparation. The maximum scheduled native work is 16*64, not a larger native
quantum. Discovery, installation, admission, slow memory and traps occur outside
the lock. No selector assigns or reconstructs guest registers.

The host runtime validates each call's progress before aggregating it. Aggregate
`MemoryContext.Total`, `MemoryTotal`, loop counters and `Remaining` describe the
whole schedule; `Retired` is still the last leaf's precise progress. Ready
publication generations use the existing `NativeLink.Reserved` field only for
host scheduling, not as a native entry/admission proof. Within this
single-threaded schedule only native non-executable data stores can occur;
there is no host mapping/code mutation, so the initial executable-context proof
remains valid. Context checks resume at the next engine entry or slow path.
Aliases do not grant concurrent mutation support.

Cold native maps now evaluate deferred predicate/selection/flag DAGs into
private spill slots, once per value per exit. Reverse lifetime propagation keeps
all leaf operands live; any loop-phi or ordinary data use remains hot. Fused
branches produce their own immediately consumed flags, including explicit
boolean inversion. Addition HI/LS retains the correct carry/zero combination.
The amd64 dispatcher similarly consumes its own budget SUB flags immediately.
No optimization relies on flags surviving an unrelated operation.

Loops keep separate call-local read/write page facts. A full page-number hit
reuses the same call's canonical-address, backing and permission proof, not a
memory value. Width/boundary checks remain per access; new facts require complete
validation; stores still check generation overflow before mutation. Facts reset
on every native call. Scratch grew by 32 bytes inside the unchanged 17,408-byte
frame and 64 KiB private stack. No persistent native target-pointer cache is used.

## Bounded trace and proof refinements (5000-score optimization pass)

Cold discovery tries both legal direct branch edges, with at most 256 attempts.
It selects a cyclic path when found, otherwise a bounded acyclic path through
already-native, same-context blocks with executable dependencies. It does not
fetch or decode new pages. Every internal edge still has a precise native guard;
all constituent code-page dependencies participate in invalidation. Back edges
are preferred, otherwise fallthrough is considered first. Direct BL and observed
indirect-target stitching experiments were discarded.

When neither final direct successor can return to the root, the region's native
terminator is constant false. This permits late input definitions and linear
allocation without loop-phi liveness. Cyclic read-only bodies can reload an input
from architectural state instead of retaining it across the iteration only when
its final value and every exit-map value are unchanged. Bodies containing a
memory store retain the conservative input convention.

Sole low-word arithmetic consumers can select 32-bit native arithmetic. Sole-use
unsigned-load / XOR-sign / subtract-sign chains can select a guarded signed
native load, optionally zero-extending its low word. Raw/intermediate users
prevent unsafe elimination. Signed result aliases inherit every ordinary,
checkpoint and delayed narrow-consumer lifetime. Independent full-width and
low-word users remain distinct. No guest flags or benchmark identities are
hardcoded by these transformations.

`PrepareTargetLink` validates cold installed-entry admissions and publishes a
bounded arena-owned table of 1024 32-byte proofs: full PC, installed offset,
exact count, relative body offset and ABI family. It adds 32 KiB of host metadata,
not guest RAM or executable capacity. Proofs retain no executable address.
`MemoryContext.PreparedTargets` at offset 67688 borrows that table only while the
arena lock is held, and is cleared before unlock on success or failure. Native
selection still compares public full PC, entry and exact count and checks the
remaining budget. Private proof construction establishes owner, alignment,
state shape, family and body bounds. A collision or mismatch takes the original
fully checked admission path. Closed owners and cross-owner contexts cannot use
or revive the borrowed proofs.

The 5000-score target is **not met**. Valid-duration results and all discarded
trials are recorded in `aarch64-coremark-score.md` and the 5000-pass evidence.

## Resumed bounded scheduling and precise inherited exits

`LinkedCall.CallQuanta` uses a typed view of the checked context prefix. The
runtime verifies field offsets and sizes, including the GC-visible page pointers.
It holds arena serialization for at most sixteen separate native calls; each
call returns to Go, validates progress, and installs a fresh budget at most 64.
It is not an enclosing assembly loop. Descriptor readiness and full PC tags are
checked between calls. Both borrowed views clear before the arena lock releases,
on success and error. The generic selector-based batch API remains available.

Dispatcher retirement is reconstructed at its cold exit from its private initial
budget minus validated remaining budget, plus its original aggregate total.
Invalid progress never reduces the remaining budget or credits a bad prefix.
Exact memory-prefix accounting remains at each validated transition.

A loop exit map's missing slot means untouched on the first iteration only.
If that slot is assigned later in the body, an early exit in another iteration
must inherit its previous final value. Private captures save the hot leaves of
that final value's SSA DAG before phi moves; deferred flags/selects are computed
only if an exit needs them. No uninitialized capture is read on the first
iteration. Captures belong to the checked frame, not guest memory or persistent
native-call state. This fixes the demonstrated unread-output fault regression.

### Atomic paired RAM operations

Checked memory-load width 16 is accepted only with its immediate structural
`PairHigh` (record 36) second result. Both ordinary 64-bit host reads follow one
complete 16-byte admission; the high result uses a distinct private SSA spill.
The pair counts as one logical guest-memory instruction. `MemoryPairStore`
(record 37) carries address, first data and an encoded third SSA data operand;
its complete 16-byte boundary, mapping, permission and generation checks precede
one generation allocation and both stores. Executable writes and crossing-page
pairs still exit before mutation. These structural operations are rejected by
pure emitters. Frame peak rejection remains unchanged; region width and the
foreign-call budget are independent controls as documented below. Whole-block and cyclic tests cover all budgets, inherited
second halves, malformed IR, permission failures and generation exhaustion.

Completed loop exits can specialize normalized boolean predicates and their
constant/selection aliases using the already-checked branch. Early fault maps
are never specialized from a terminator they have not executed. Arbitrary
nonzero conditions retain their full values.


The prepared Go quantum scheduler may retain cumulative native-generated counters
across its separate calls. Each call still has a fresh <=64-instruction remaining
budget, resets leaf retirement, and returns to Go before any next call. Go checks
monotonic total/memory counters, delta <=budget, exact remaining=budget-delta, and
memory delta<=retirement delta before crediting that quantum. An invalid quantum
leaves the last validated aggregate credited, and all borrowed admission views
are cleared before releasing the arena lock. The batch remains <=16 actual calls.

## Experimental checked direct edges and one-word transfer

`EngineConfig.DirectChaining` (Linux CLI `-native-chains`) is **false by default**.
The valid-duration trials do not establish a throughput improvement; see the
chaining follow-up in `aarch64-coremark-score.md` and its raw JSON evidence.

`CompileLoopChained` accepts at most eight aligned cold-known exit PCs. Every
successful edge refreshes both full PC tags, the public entry and exact count,
the private ABI family, live code base and fresh remaining-budget admission.
Only the shared-loop family can tail-jump with the original return address.
Misses, collisions, other families and invalidated descriptors return to the
unchanged dispatcher. The final descriptor/count/remainder are the successor's,
so the ordinary return continuation validates and credits only its final leaf.
A source body accounts its precise memory prefix before an edge; earlier progress
is charged once, and the quantum is never replenished. Access/trap exits never
attempt a direct edge. Successful partial side exits retain their precise maps.
Final and internal conditional successors are collected cold, deduplicated and
bounded by eight. Excess or indirect/call successors use ordinary dispatch.

On AMD64, a chained body with no memory operations and a nonzero PC slot may
retain architectural slot zero in **R10**. That register is excluded from SSA
allocation and continuation scratch; direct-edge progress uses RSI instead.
Its ordinary installed start initializes R10 using `MOV R10,[R11]`. The interior
entry three bytes later is **not a public installed entry**. Only an arena-owned
prepared proof's additional family-word bit zero permits a transferring source
to skip the initializer. Generic dispatch always enters the ordinary start.

An arena-owned cold map records transfer-capable public offsets; it is bounded
by installed images/code capacity, stores no executable addresses and is freed
on Close. `PrepareTarget` folds that mark into the existing 32-byte proof; no
entry-record/table, guest-memory, frame, private-stack or code-capacity limit is
increased. There is no guest- or descriptor-supplied transfer flag.

Architectural input loads and exit-map stores for slot zero become R10 moves.
Its value is stored back before every trap, budget exit, missed edge, incompatible
successor or ordinary dispatcher/Go return. Precise inherited-output captures
and cold exit DAGs remain active. Other architectural words still materialize.
Memory-containing bodies are deliberately excluded: trusted backing pages can
alias the state allocation, so postponing state stores there needs an additional
alias/effect proof. A transferred pure value is flushed before a nontransferring
loop; other ABI families use the fully materialized dispatcher continuation.

Cyclic true-terminator budget exits cannot admit their own body again, so they
bypass impossible direct-edge probes. Successful branch/partial side exits share
one checked continuation. Both admission views are still borrowed under the same
arena lock and cleared before unlock. Cross-region register transfer is not
itself an enlargement of either execution limit. Legacy calls retain their
<=64 ceiling; the independent wider region/session ABI is documented below.

Execution evidence is Linux/amd64. Arm64 direct edges are emitted but not executed
on Arm64 hardware here; the register-transfer extension is AMD64-only.

## Rejected alias-safe memory-transfer prototype (October 8, 2026)

A measured AMD64 prototype extended R10/slot-zero transfer to memory bodies.
It guarded every translated physical byte range against that word, plus each
store's clock/epoch pointers. A hit materialized before access and reloaded after
store; shared-page and invariant-address paths passed through the same checks.
Whole 16-byte widths were checked before either paired half. The local alias
flag used loop-private displacement -144 (physical FP-272 after the unchanged
128-byte bias), below the preserved-register and fact areas and above spills.
State/context overlap selected a duplicate materialized entry body; that
fallback retained the public entry and original continuation ABI. No pointer
proof survived a host call, and no guard or resource bound was weakened.

The scalar alias oracle, cyclic-cache comparisons, native and emulator tests,
and 30-second unchanged preflight passed. Nevertheless two valid-duration
CoreMark runs fell to approximately 3,361 performance iterations/second against
3,805 default and 3,743 genuinely enabled pure-only control. The source/test
snapshots are in `sandbox/rfe-memory-transfer-prototype/`, not the retained code.
See `aarch64-coremark-memory-transfer-results.json` for raw timing and selection
corrections to the earlier final opt-in label. This tests a conservative per-access
prototype, not liveness-selected multi-register allocation or an owner-bound
stable nonalias contract; its failure is not evidence that every such design fails.


## Independent optimized regions and foreign sessions (October 8, 2026)

`MaxInstructions` still bounds cold blocks and `Step` (default 16).
`MaxRegionInstructions` independently bounds optimized hot traces (1..256,
default 256). Counts no longer occupy four family bits: installed and admitted
counts are full fields, and prepared proofs still bind full PC, entry, state
shape and exact count. Wider regions account their own precise dynamic memory
progress; only ordinary <=16-instruction leaves index the 17-byte prefix table.
The existing 2048-record IR bound, 17,408-byte shared frame, 64 KiB private stack,
code capacity and resource gates are unchanged.

`MaxNativeInstructions` independently bounds each `Run`/`RunQuanta` foreign
session (1..65,536, default 65,536). A session can stop early on a missing target,
a remaining budget shorter than a region, or an architectural slow exit. It
never rounds a caller budget up or retires an uncommitted faulting instruction.
The caller can handle remaining tails with the ordinary short block/Step path.
Cancellation and host work can be observed between sessions; this is an
instruction bound, not a guaranteed wall-clock response time.

Long sessions currently require **Linux/amd64 with cgo**. A separate System V
trampoline is entered through an ordinary cgo call, saves C callee-owned
registers on the C stack, switches to the existing private native stack, and
restores the C stack before returning. It has no Go callbacks, runtime-private
linknames, or bypass of cgo pointer checks. Other hosts/no-cgo builds and contexts
whose RAM/clock/epoch aliases architectural state or the context use the old
<=64 boundary. The legacy assembly budget was not raised.

The arena owns dedicated pointer-free state and ABI-image allocations, reused
under its existing lock. An embedded CPU array or `MemoryContext` must not be
passed directly to C: enclosing allocations can contain unrelated unpinned Go
pointers. Every native-reachable allocation is explicitly rooted and pinned
for the call: ABI image, state image, private stack, current admission metadata,
proof table, authoritative descriptor field, backing pages, clock and epochs.
Only the 2,136-byte memory/budget prefix is copied per call, not the 64 KiB link
table. The new scalar at context offset 67,696 selects the borrowed descriptor
base on the session-private image. Zero selects the legacy inline table.
Native selection still checks the authoritative public PC/entry/count every
time; copying descriptors cannot hide invalidation or guest-visible aliases.

Committed state and exact progress are copied back before return. Borrowed
addresses are cleared before unpinning and the arena lock is released afterwards.
No pin/view survives the call, and Close discards reusable storage and invalidates
old handles. Aliased contexts use original state/context storage rather than
silently detaching aliases through a shadow. Native host contexts remain trusted,
not a security sandbox.

Zero new fields in old `EngineConfig` literals preserve compatibility: region
width defaults to `MaxInstructions`, session budget to 64. The Linux CLI exposes
`-region-instructions` and `-native-instructions`; direct chaining remains a
separate, disabled-by-default experiment.

Tests cover scalar retirement at widths 16/17/32/64/65/128/256, tiny/odd/full slice
budgets, late access faults, exact count rejection, full-PC/proof collisions,
clear/close cleanup, embedded Go owners, RAM/clock/epoch/context aliases, and
concurrent callers plus GC with `GOMAXPROCS(1)`. Only Linux/amd64 execution has
been measured here; this does not claim a validated long-session Arm64 bridge.
CoreMark controls and raw ticks/CRCs are in
`aarch64-coremark-session-results.json` and `aarch64-coremark-score.md`.


## Four-way native links and session-scoped admission reuse (October 8, 2026)

The current native link directory has **256 sets of four ways**, retaining the
existing 1024 public slots. Its set index is `((PC >> 2) ^ (PC >> 12)) & 255`;
ways are separated by 256 records. Full 64-bit PC tags and nonzero instruction
counts remain mandatory. Empty zero-PC records cannot hide a valid later way.
Publication updates an existing full-PC match, otherwise uses an empty way or
bounded round-robin replacement. Clear invalidates every way; owner changes
still invalidate the complete public directory.

Private prepared proofs use the same set/way scheme, but their placement is
independent of public placement. Both tables are probed by full PC. The private
proof stride is now **64 bytes**, superseding earlier 32-byte layout descriptions:
PC, entry, instruction count and body/family retain offsets 0, 8, 16 and 24;
offset 32 stores a checked-session generation and offset 40 a validated public
**relative byte offset**, never a retained host descriptor pointer. The private
proof table grows from 32 KiB to 64 KiB; public-table capacity, code-arena limits,
shared frame, private stack and execution budgets do not grow.

A foreign session may reuse a descriptor admission only after its physical-alias
checks establish that guest RAM, state, clock and epoch effects cannot modify
the public context. A nonzero arena-serialized generation is assigned to each
such invocation. The dispatcher freezes it at entry in its private frame slot
FP-120. This slot is outside leaf SSA and translation facts. It must never reload
cache authority from guest-visible context during execution: an aliasing caller
could otherwise overwrite a public scalar to resemble an earlier generation.

On first use, prepared dispatch checks full public PC, exact count and entry
against the private proof, then records its generation and public relative
position. Repeated uses in that invocation still check full private PC, presence,
family and **fresh remaining budget**, but may skip the already-proven immutable
public comparisons. Ordinary leaves use the validated public way for their
precise memory-prefix accounting; loop regions continue accounting their own
memory progress. No state value or memory content is cached by this mechanism.

Every later session has a fresh token, including when another MemoryContext uses
the same arena. Generation wrap clears all cached generations before reuse.
Republishing a private proof resets its cached generation. Legacy, aliasing and
unsupported-host calls begin with generation zero and cannot enable reuse through
subsequent guest writes. All borrowed public fields are cleared by session cleanup.
Native mappings, code and proof publication remain serialized by the arena lock;
concurrent host mutation of a borrowed context is not supported.

Regression coverage includes mixed pure/memory/loop execution through all four
colliding ways with deliberately different public/proof placements, exact tiny
and 65535/65536 budgets, scalar state and memory accounting, empty-PC holes,
replacement/clear/full tags, cross-session mutations, context changes, forced
64-bit generation wrap, and guest writes that attempt to enable old authority
while invalidating a successor. Existing fault, direct-chain, invalidation,
GC/scheduling and compatibility tests remain in force. Native execution was
validated on Linux/amd64; arm64 emission was updated but not executed here.

`native_sessions`, `link_miss_stops`, `budget_stops` and `other_stops` diagnose
linked-engine invocations and their successful-return causes. With long sessions,
one invocation is one foreign session; legacy grouped quanta remain multiple
short native calls within one invocation. Missing-link classification is based
on the returned full PC, not on a guessed dispatch-time percentage.
