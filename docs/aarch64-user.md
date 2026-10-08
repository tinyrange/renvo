# AArch64 user-mode RFE and tiered recompiler

> Current benchmark summary: `aarch64-coremark-score.md`. Dated sections below
> retain development history; older results and proposals are not current claims.
> Historical raw trial/profile filenames refer to local artifacts not included
> in this PR. Current score evidence is checked in alongside the summary.

## Goal and current status

The first performance goal is a repeatable win over a pinned QEMU Linux-user
build on the **same AArch64 CoreMark ELF**, initially on Linux/amd64. Guest
AArch64 on host AArch64 is a separate result, not interchangeable evidence.
Linux/amd64 is the initial benchmark target, not a restriction on portable
interpretation or the existing native emitter's supported hosts.

This document separates implemented bounded native loop traces from the proposed
general optimizing-region engine. The current implementation runs the pinned freestanding CoreMark
workload with matching CRCs in every tier, but **does not establish a QEMU
speedup**. The October 8 confirmed working tree yields native
trials of **5,348.42–5,361.57** versus a QEMU
comparison of **7,546.54**, using the same 100,000-iteration ELF. These are
sequential unpinned trials, not a new controlled median. See `aarch64-coremark-score.md` for
full reporting context and evidence. It is not a
complete AArch64 CPU or a complete Linux user-mode implementation.

| Component | Implemented | Remaining milestone |
| --- | --- | --- |
| Packaging | Authored `aarch64.rfe` and `linux-arm64-user.rfe` | Reuse the engine across additional guests |
| CPU | Integer subset, flags, branches, scalar loads/stores and register pairs | Broader ISA coverage; FP/ASIMD, atomics, system registers |
| Execution | Interpreter, hot pure IR, native pure/scalar-memory blocks, bounded native linking and cyclic loop traces | Pairs/literals, division and general CFG regions |
| Optimization | Folding, masks, state forwarding, value numbering, direct register emission, rich integer IR, native loop phis/exit maps, cross-block trace residency and call-local translation facts | General region CFG/SSA, SCCP, alias/effect-aware memory optimization and residency outside bounded traces |
| Linux personality | Static ET_EXEC loading, initial stack, bounded anonymous memory and syscall slice | TLS, libc startup, signals, file ABI and dynamic linking |
| Performance | Pinned CoreMark port, CRC agreement, diagnostics, guarded interpreter cache and valid-duration native/QEMU scores | Full host characterization and repeatable QEMU win |

The editable archives are the source of truth. There is no parallel checked-in
Go implementation. The source-package test extracts the archives, resolves the
CPU dependency offline, verifies canonical formatting and executes their
embedded tests. RFE package code is trusted host code, as with the PDP-11 RFEs.
Guest permission checks and resource budgets do **not** make a security sandbox.

## Packages and responsibilities

- `emulators/aarch64.rfe`: CPU state, normalized decoder, reference execution,
  pure instruction lowering and initial tier controller. Its import identity
  is `renvo.dev/emulators/aarch64`; there is no executable entry.
- `emulators/linux-arm64-user.rfe`: 64-bit paged address space, ELF loader,
  initial process state, syscall personality and `Main`. Its import identity is
  `renvo.dev/emulators/arm64user`; it depends on the `aarch64` archive.
- `internal/rfe/runtime`: existing architecture-neutral pure IR and native
  emitter API. Value numbering lives here, not in AArch64-specific compiler
  code. This also benefits other clients of the pure builder.

A64 encodings often have more than the lowering DSL's supported 16 variable
opcode bits. This increment does not relax that validation limit. An explicit
A64 decoder produces normalized instructions consumed by both the interpreter
and the direct IR lifter. The interpreter and lifter implement arithmetic
independently, so tests can catch lowering mistakes; architectural vectors
remain necessary because both paths share decoding.

The initial tier controller is guest-local. When a second guest needs its
profiling, invalidation and promotion machinery, extract those mechanisms into
a shared engine with a guest adapter. Do not introduce a nominally generic
framework before its memory/effect and exit contracts are understood.

## Experimental direct chaining

`EngineConfig.DirectChaining` or the Linux CLI's `-native-chains` enables checked
direct tail edges and AMD64 one-word transfer between eligible memory-free
regions. It is disabled by default: valid-duration trials do not demonstrate a
throughput improvement. Chaining does not itself enlarge execution limits or
retain values across memory regions. Independently, the optimized-region and
supported foreign-session ceilings are now configurable up to 256 and 65,536;
cold blocks/Step remain at 16 and conservative native calls remain <=64. See `rfe-native-abi.md` and `aarch64-coremark-score.md`.

## Architectural contract

`CPU.State` has 34 unsigned 64-bit slots:

| Slots | Meaning |
| --- | --- |
| 0–30 | X0–X30 |
| 31 | SP; never the backing storage of ZR |
| 32 | Guest PC |
| 33 | NZCV in architectural bits 31–28 |

Register 31 is interpreted per operand role. W writes zero-extend, unsigned
arithmetic wraps at its architectural width, subtraction carry means no borrow,
and host condition flags are not silently substituted for guest NZCV.

Implemented instruction groups include:

- ADD/SUB immediate, shifted and extended register, with and without flag updates;
- ADC/SBC and flag-setting variants;
- shifted and immediate AND/ORR/EOR/ANDS, including inverted shifted operands;
- MOVN/MOVZ/MOVK and ADR/ADRP;
- UBFM/SBFM/BFM, EXTR and common bitfield/shift/sign-extension aliases;
- variable LSL/LSR/ASR/ROR, bit/byte reversals and CLZ/CLS;
- MADD/MSUB, widening signed/unsigned multiply-add/subtract, SMULH/UMULH
  and conditional-select variants in all tiers; UDIV/SDIV in the interpreter;
- conditional compare/compare-negative with register and immediate operands;
- B/BL, B.cond, CBZ/CBNZ, TBZ/TBNZ and BR/BLR/RET;
- scalar integer loads/stores: unsigned offset, unscaled, pre/post-index,
  extended/scaled register offset, signed loads and literal loads;
- offset and pre/post-index integer register pairs, including LDPSW;
- NOP and the explicit PACIASP/PACIBSP, AUTIASP/AUTIBSP and BTI compatibility
  HINTs as NOPs for the base CPU without PAC/BTI; structured SVC exits.

This does not implement pointer authentication or BTI enforcement. Non-HINT
authentication instructions remain unsupported; the ELF auxiliary vector does
not advertise PAC/BTI capabilities.

Reserved encodings, unsupported groups and constrained-unpredictable operand
overlaps are rejected explicitly. Non-temporal pairs, unprivileged loads,
FP/ASIMD, exclusives/LSE, barriers and general system-register access are not
implemented. Nothing is treated as a successful no-op merely to advance a test.

Fetch faults retain the faulting PC. Memory accesses validate before updating
registers or writeback bases. Pair stores use a full-range precheck in the
single-threaded memory implementation; this is a deterministic implementation
choice, not a claim that pair accesses are atomic on real hardware. An SVC is
retired and advances PC before returning to the personality. Other rejected or
faulting instructions do not retire. There is no guest signal frame yet: faults
terminate the current run with a structured diagnostic.

## Initial execution tiers

The modes are `interpreter`, `ir` and `native`. Native mode includes the lower
tiers; a native mapping failure is recorded and falls back to portable IR.

Defaults:

- Build a bounded profile for up to 16 consecutive lowerable instructions, or
  one guarded interpreter-only instruction word.
- Interpret until eight visits to a profiled block entry.
- Build optimized pure IR at the IR threshold, not on the first cold visit.
- Attempt native compilation at 64 visits.
- Keep at most 4,096 profiled block entries and an 8 MiB native arena.
- Fall back when either cache or compilation limits are reached.

Pure blocks include integer arithmetic, flag updates, bitfields, moves,
PC-relative addressing, multiply, conditional select/compare, variable shifts
and reversals/count-leading operations. A block can end with a compiled B/BL,
B.cond, CBZ/CBNZ, TBZ/TBNZ or BR/BLR/RET. The branch computes its successor
PC and link register in IR/native code. Native mode can then enter a published
successor through the bounded native dispatcher described below. Missing or
invalid destinations return to host dispatch with the branch already retired.
Scalar memory has a checked native path; pairs, literals, traps and division
still return to architectural execution. Modular multiply uses a shared
IR operation and the existing native multiply emitters. Variable shifts/rotates use explicit width-aware IR and native instructions.
Arithmetic/logical flag computation and derived conditions use compact native
lowering; guest NZCV stores remain observable at exits. The portable IR evaluator
is a correctness/portability tier, **not an assumption of better throughput than
the interpreter**. Promotion thresholds need measured tuning.

The Linux personality supplies optional versioned code-page guards. A block
records every 4 KiB page covering its instruction words, with an address-space
identity and a monotonically allocated page generation. Hot entry checks those
stamps and execute permission rather than rereading every instruction. Writes
(scalar, buffer and pair), permission changes and same-address remapping cannot
reuse a stamp; unrelated data pages do not invalidate the block. Generation
exhaustion fails before mutation, and aliased copies share the generation clock.
Loader-only writes occur before the address space or any stamp is exposed.
An optional single-threaded `CodeContextMemory` adds a whole-address-space
executable stamp. An unchanged context reuses previously validated dependencies;
a changed context requires full page revalidation. Only executable content,
execute permissions and executable mapping changes advance it, not ordinary
data stores. Scalar, buffer and pair writes, map/protect/unmap and aliased memory
all preserve this contract. Executable unmap also fails transactionally if a
fresh generation cannot be allocated. Custom page-version-only memories remain
supported, with their original per-entry page validation.

Other memory implementations remain supported through full instruction-word
revalidation. A versioned block never assumes an unversioned replacement memory
is the same address space. Byte or stamp changes, or revoked permission,
invalidate the cached block before it executes.
Obsolete native entries remain unreachable in the bounded append-only arena;
there is no unsafe reclamation. A bounded 1,024-slot direct lookup accelerates dispatch without replacing the
profile map or relaxing its budget. Collisions fall back to that map; invalidation
clears both lookup paths. Native translations specialize their guarded entry PC,
so PC-relative addresses and sequential updates fold at translation time.
The portable tier prepares an immutable validated copy of IR and reuses an
engine-local 2,048-value evaluator frame without per-entry validation/allocation.
This machine, like the guest engine, is single-threaded and non-reentrant.

A compiled block is used only when the remaining
instruction budget covers it completely; otherwise execution takes one
interpreted instruction. Interpreter-only entries use the same guards and then
execute their cached word through architectural semantics, avoiding a duplicate
fetch. They share the existing bounded cache and never bypass memory faults,
retirement accounting or unsupported-instruction diagnostics. Cached scalar
and pair instructions dispatch directly to the original architectural handlers,
without repeating the pure/extra decoder classification. Other instructions
retain `CPU.Execute`, including precise trap retirement.

`Engine.Run` uses a bounded 64-instruction native dispatch quantum when the
memory supplies both native descriptors and a non-nil executable-context
identity. A 1,024-slot full-PC-tagged table publishes only entries validated in
the current context. Its offsets are invalidated whenever the executable context
or arena owner changes. Native transitions check PC alignment, whole-block
budget, offset alignment/range, installed leaf entry and exact state shape.
Dispatcher entries cannot be link targets. Cold/missing successors return to Go;
compilation and publication remain outside the arena lock. `Engine.Step` retains
its single-block API. Other memories retain the guarded host batching fallback.
No background compilation or general CFG optimizer is present. Hot direct
self-loops and cyclic traces of already-validated native blocks compile one
iteration with loop phis and native cold exit maps. Their combined static size
still fits the 16-instruction ceiling. Guest values remain resident across the
chosen internal branches and iterations; state is reconstructed natively on
exits. Branch side exits can continue native dispatch; mapping/trap/access slow
paths return to architectural execution before the pending instruction. The
64-instruction quantum, Step leaf API and short-budget leaf entries are intact.
Trace dependencies include every constituent code page. `regions`,
`region_exits` and `loop_iterations` report compiled traces, successful exits
and iterations without a Go transition.

High multiply, leading counts, byte/bit reversal and carry/NZCV remain rich IR
until native instruction selection. Ordinary SSA instructions emit directly in
allocated registers. Exit-only flags/selects are computed in cold native stubs,
with their operands kept live; a flags value needed by a loop phi stays hot.
Within a serialized loop call a checked one-page translation fact can be reused.
Variable addresses keep range, width, tag and permission checks. An address proven
invariant across every loop phi gets an exact-access fact after those guards
succeed once; mapping/permissions cannot change inside this single-threaded call.
Facts cache translations, never memory values. Every store still checks live clock
overflow and updates the actual epoch/data. Facts do not survive a native call. Native execution evidence currently covers Linux/amd64,
not the arm64 emitter on arm64 hardware.

### Native scalar memory and precise exits

Native mode profiles mixed pure/scalar blocks up to the same 16-instruction
limit. Effects remain architectural until native promotion; the portable IR
contract remains pure. Ordered memory IR retains all accesses, even a load to
ZR, plus state/progress checkpoints before each potentially faulting instruction.
Memory loads are not value-numbered across loads/stores. These are effect roots,
not removable computations or permission checks that can move past stores.

The Linux memory owns a shared 64-slot native page table with typed host pointers
that keep page bytes, the page generation and global clock alive. Accesses
check the full guest-page tag, 48-bit address range, page boundary, current
permissions and live data pointer, except for the already-proven invariant
accesses inside one loop call described above. Stores additionally require a non-executable
writable page and a non-exhausted generation clock; they update the actual page
stamp before data. Protect/unmap/remap invalidate descriptors across aliases.

Cross-page operations, descriptor misses, executable stores and failed SP guards
exit before the access or writeback. The committed prefix and its retirement are
accounted once; architecture executes the slow/faulting instruction once. It
never replays earlier stores. Executable stores cannot continue into a previously
published native successor. The native loop checks partial progress before
indexing the memory-prefix counter and returns on every slow exit.

The arena lends its code base and installed-entry metadata only while holding
its original mutex, clears that view on return, and retains the private stack
and host pointer graph throughout. No unchecked executable address escapes. Dispatcher calls may use an opaque,
once-validated owner-bound handle; every call still checks owner lifetime and
state/context shape under that mutex and borrows the current append-refreshed
view. Native leaf admission checks are unchanged.
`native_memory`, `memory_exits` and `linked` report scalar retirement, precise
slow exits and instructions retired inside the native dispatcher, respectively.


The shared native arena now records each installed entry's state size in a
bounded 16-byte-offset-indexed metadata table. Alignment, installed-entry, shape,
closed/broken-arena checks and private-stack lifetime remain enforced under the
original mutex. The table uses an eight-byte record per 16-byte code offset:
installed state/family metadata, a pointer-free immutable link-admission key
and a checked relative internal-body offset.
Cold publication validates the exact state/body shape once; native selections
still compare every public count and the dispatcher's shape against that key.
Unadmitted entries retain the full native admission path. Keys are append-only,
rooted during calls and discarded on Close; forged counts cannot authorize a
leaf or loop. The metadata's logical payload is at most 4 MiB for the default
8 MiB code arena (previously 2 MiB), excluding Go slice-capacity overhead.
No unchecked entry handle escapes a batch.

Linked entries now borrow one 17,408-byte dispatcher-owned frame on the existing
64 KiB private stack. The 128-byte dispatcher header is disjoint from SSA/phi
scratch, register saves and translation facts. Shared bodies inherit the state
base and persistent context slot; pure spills reserve that slot as well. Pure
and memory images retain checked standalone compatibility entries. Legacy
framed blocks can mix with shared bodies, including legacy state-base scratch
clobbers. Loops preserve only the owned registers they actually use. See
`rfe-native-abi.md` for the ownership/layout contract and remaining cross-block
guest-value materialization.

This baseline intentionally trades speed for simple invalidation and precise
state. It is not the final strategy for beating QEMU: slow-path memory
accesses, dispatcher instruction count and conservative instruction selection remain
profiling targets. `code_reads`, `version_checks` and `context_checks` in `-stats` count translation
discovery/revalidation, not all architectural fetches. The page-guard regression
checks 100 entries to a two-page block use 100 executable-context checks without
further page-map or instruction-word reads. Versioned memories without the
optional context API still perform 200 page-version checks; unversioned memories
still revalidate instruction bytes. These are mechanism tests, not scores.

## User-mode memory and ELF contract

The address space is little-endian, restricted to guest addresses below 2^48,
with 4 KiB pages and separate read/write/execute permissions. Guest RAM is
limited to 64 MiB; this is **not** a cap on total host RSS. A shared 64-slot
page lookup cache avoids repeated mapping-table hashes for common accesses.
It caches page ownership, never permission: every access checks current page
permissions. Memory aliases share the cache, unmap removes cached ownership,
and remapping cannot expose detached page contents. Fetches may read
execute-only pages; data reads may not. Cross-page stores, protection changes
and pair accesses validate the complete range before committing.

The ELF parser reads only a bounded program table, never the section table.
It requires ELF64, little endian, EM_AARCH64, ET_EXEC, ordinary program headers,
a mapped readable program table and an aligned executable entry. It validates
file ranges, address arithmetic, segment alignment, overlapping byte ranges
and allocation limits. Disjoint segments may share a page with the union of
their permissions. PT_INTERP, PT_DYNAMIC, PT_TLS and an executable-stack request
are rejected. This intentionally excludes ordinary dynamically linked programs
and many statically linked libc programs until startup/TLS support is added.

The initial stack has:

- a 16-byte-aligned SP, argc, argv, NULL and an empty environment;
- an auxiliary vector with program headers, page size, entry, conservative
  identity/capability values, exec filename and fresh AT_RANDOM bytes;
- at most 256 arguments and 64 KiB of argument strings;
- a 1 MiB non-executable mapping with unmapped space below it.

Initial guest environment variables are not inherited from the host. The
personality exposes no guest file-opening syscall or host syscall passthrough.
The CLI explicitly reads the requested host ELF; guest writes use copied,
bounded buffers and only the supplied stdout/stderr writers.

Implemented syscall numbers:

| Number | Operation | Initial restriction |
| --- | --- | --- |
| 64 | write | stdout/stderr only; short transfers capped at 64 KiB |
| 93, 94 | exit / exit_group | One process and one thread |
| 113 | clock_gettime | Realtime and process-relative monotonic elapsed clock |
| 172, 178 | getpid / gettid | Virtual ID 1 |
| 214 | brk | At most 16 MiB beyond the initial break; old break on failure |
| 215 | munmap | Page-aligned address, rounded length |
| 222 | mmap | Anonymous private, zero hint, no fixed mappings |
| 226 | mprotect | Mapped page ranges only |

Unsupported syscalls return `-ENOSYS`, not fake success. SVC must have immediate
zero for this Linux personality. Clock values derive from real host time, never
retired instruction counts. The monotonic clock's epoch is process load time;
it is suitable for elapsed intervals, not a claim of host-uptime ABI fidelity.

## General optimizing region architecture: remaining design milestone

The bounded loop/trace phis, native exit maps and call-local translation facts
described above are implemented. The following outlines their extension to a
general CFG and a broader effect-aware optimization pipeline, not additional
features already validated by the current trace implementation.

### Low-latency baseline JIT

Extend the IR with explicit guest-memory operations, guards, exits and branches.
Start with typed sizes, signed extension, byte order and permission checks.
Implement direct edges with unlinkable metadata; keep a mapped-code dependency
set for each translation. Unsupported instructions take explicit helper exits.
Measure translator latency as well as generated execution cost.

The immediate objective is to remove repeated dispatcher round trips and
Go helper calls from hot ordinary loads/stores, not to accumulate optimization
passes while those costs remain dominant.

### Hot regions and guest-state SSA

Collect block/edge/backedge counters under a fixed profiling budget. Select
bounded regions around hot loops; terminate at syscalls, unsuitable effects or
region-size limits. Construct a CFG with explicit side exits and SSA phi nodes
for guest registers, flags and memory effects. Verify dominance and predecessor
invariants before optimization and after transformations that change the CFG.

Keep live guest values resident across guest branches. A region entry has
explicit assumptions; each guard exit has a reconstructible architectural state
map. Background compilation is a later optimization: first make synchronous
promotion deterministic and observable.

### Optimization pipeline

Run separate, measurable passes with optional diagnostic dumps:

1. Canonicalization and local width/known-bit propagation.
2. Sparse conditional constant propagation and unreachable-block removal.
3. Copy propagation and demand-driven NZCV elimination.
4. Dominator-scoped value numbering and pure common-subexpression elimination.
5. Effect-aware load forwarding and redundant-store elimination.
6. Dead guest-state-store and dead-value elimination using exit-state demands.
7. Loop-invariant motion under fault/effect constraints.
8. Strength reduction and guarded specialization where actually profitable.
9. Host instruction selection, register allocation, scheduling and hot layout.

Each memory operation has ordering and fault properties. An unused load can
still fault; a store can modify executable code; a helper can observe guest
state. Do not erase or move those effects using pure-expression rules. Alias
analysis starts conservatively, with explicit unknown-memory barriers. Restrict
speculation to operations with a defined guard and recovery strategy.

### Precise exits, invalidation and runtime ownership

Maintain host-PC-to-guest-PC metadata for faulting native instructions, plus
state reconstruction locations for registers, spilled values and deferred
flags. Internal safepoints must satisfy the retirement budget and later signal
polling contract even when direct edges form a tight loop.

The initial single-threaded personality now implements mapping identity and
code-page generations. Preserve participation by **every** mutation path when
adding native stores, signals, loader reuse or shared memory. Loader-only raw
copies must never become a live mutation path. Mapping an executable page back
at the same address must not resurrect an old translation. Never reclaim code while
it can still be reached through a linked edge or a running thread.

Use the existing native arena's OS mapping and instruction-cache adapters;
respect W^X policy and the Go/native ABI. Native guest addresses are not Go
pointers. Background compilation must not retain movable/lifetime-sensitive
state accidentally. Guest threads require a separate memory-ordering, atomic,
signal-delivery and reclamation design before enabling them.

## Correctness and validation

The first tests cover literal architectural results, register-31 roles, W
zero extension, carry/overflow, sign-replicating shifts, bitmasks, signed branches,
failed writeback, register pairs, extended offsets and reserved encodings.
Fixed-seed multi-instruction tests compare the independent interpreter with
portable IR and actual native execution. Supported native hosts must execute
native blocks rather than silently skip that requirement.

End-to-end tests load synthetic ELF containers through the real loader, execute
write/exit programs and hot loops in all modes, validate argv/auxv/BSS, exercise
permission and allocation errors, and enforce exact retirement limits. Cache
tests change code and revoke execute permission after native promotion.
The shared IR value-numbering test checks old and new versions of a modified
architectural slot, including unsigned wraparound. Additional carry tests use
arbitrary-precision mathematical results and signed bounds rather than shared
CPU helpers. All 16 condition codes are checked against an independent truth
table, and mixed instruction chains exercise flag/state SSA forwarding. A
synthetic ELF executes 14 pure instructions through the expanded compiled tier
and then exits with SVC; it is explicitly not CoreMark. Code-stamp tests cover
cross-page dependencies, scalar/buffer/pair writes, permission revocation,
address-space replacement, same-address remapping, aliased memory and generation
exhaustion. Native multiply also has direct shared-IR semantic tests. Compiled
branch tests cover signed displacement wraparound, W/X compare widths, all test
bit positions, zero-register operands, old-LR capture in BLR X30, exact budgets,
terminal-word invalidation and destination fetch/alignment faults. A separate
synthetic countdown ELF keeps all eight non-SVC instructions (including three
loop branches) in compiled execution; that is coverage evidence, not throughput.

Follow-up requirements before claiming general compatibility:

- Differential tests against pinned QEMU and, where available, native AArch64.
- Independent decoder coverage, malformed-input fuzzing and width corner cases.
- Optimizer pass-by-pass equivalence, CFG verification and failing-case reducers.
- Broader native memory forms and execution of the native emitter on AArch64 hosts.
- More workloads covering page faults, mapping reuse and linking; safe cache reclamation.
- Real compiled guest fixtures with disclosed toolchain and expected output.

Existing native mapping support includes Linux/Windows amd64 and arm64, and
Darwin arm64. A passing run on one host does not validate all those targets.

## CoreMark benchmark contract

No benchmark-specific opcode substitutions, precomputed results, recognized
payload fast paths or alternate execution semantics are permitted.

Before collecting results, check in or otherwise pin the following metadata:

- CoreMark revision, license, port source and all build flags;
- guest compiler version, ELF SHA-256, seeds, workload size, iteration count,
  validation CRCs and timer implementation;
- Renvo commit and mode/threshold/cache settings;
- QEMU revision/release, build configuration, CPU model and command arguments;
- host OS/kernel, CPU model, affinity, power/frequency policy and memory limits.

Use identical guest bytes, guest features, arguments and iterations for both
engines. Perform short diagnostic runs separately from valid scoring runs.
Use the published CoreMark run rules, including the required run duration and
validation. A freestanding port is acceptable only when its changes are
transparent, benchmark computation is unchanged and reporting requirements are
met. Do not describe an unvalidated checksum or shortened run as a score.

Measure and report both whole-process elapsed time and benchmark-loop
throughput. Compilation is charged to the engine: distinguish warming the host
filesystem/page cache from precompiling guest code. First comparisons start
both processes with empty in-process translation caches. Any persistent-cache
experiment is a separately labelled result with cache provenance.

Use at least seven paired trials in balanced order. Publish raw times,
validation output, medians and an uncertainty estimate; investigate noisy runs
rather than deleting inconvenient samples. Target a repeatable >=10% throughput
win with the uncertainty interval supporting a win, not a one-off best run.
Also report translator time, tier coverage, code size, fallback frequency and
peak RSS. Native AArch64 is a context/control measurement when hardware exists,
not the denominator for the QEMU speedup claim.

A CoreMark win is a workload milestone, not proof of general superiority.
Afterward add programs with calls, branches, pointer chasing, memory throughput,
faults and syscalls. Retain correctness and resource gates while optimizing.

## How to build and run

From the repository root, using the existing RFE tooling:

```sh
go run ./cmd/renvoemu test emulators/linux-arm64-user.rfe
go run ./cmd/renvoemu build -o sandbox/linux-arm64-user emulators/linux-arm64-user.rfe
sandbox/linux-arm64-user -engine interpreter -stats ./static-aarch64-guest arg
sandbox/linux-arm64-user -engine ir -stats ./static-aarch64-guest arg
sandbox/linux-arm64-user -engine native -steps 100000000 -stats ./static-aarch64-guest arg
```

The CLI defaults to native mode and a 100 million instruction ceiling. A long
CoreMark scoring run will need an explicitly chosen larger ceiling, not removal
of instruction accounting. The emulator is Go-hosted; hot native blocks are
emitted by Renvo. A fully self-hosted emulator remains separate work.

## Delivery sequence

1. **Implemented foundation:** archives, integer subset, static ELF, syscall
   slice, bounded pure tiers, optimizer value numbering and embedded tests.
   Follow-up integer coverage, native multiply/conditional operations and
   single-threaded versioned code-page guards are implemented as well.
2. **Implemented pinned CoreMark compatibility:** freestanding port, real
   instruction/startup/syscall closure, all-tier CRC agreement and native runs
   exceeding ten seconds for both seed sets. See `aarch64-validation.md` for
   raw evidence and the still-unmet paired performance gate.
3. **Baseline performance, partially implemented:** checked scalar native memory,
   compiled control flow, profiling, bounded native linking and page dependencies.
   Pairs/literals/division, further instruction selection and translation-cost measurements remain.
4. **Optimizing regions, partially implemented:** bounded cyclic traces, loop
   phis, cross-block trace residency and native precise exit maps are present.
   General CFG/SSA and the broader effect-aware optimization pipeline remain.
5. **Performance gate:** reproducible QEMU comparison with raw evidence and
   repeatable win. If a gate fails, profile and fix the bottleneck, not the gate.
6. **Compatibility expansion:** signals, TLS/libc, files, dynamic linking,
   atomics/threads and broader ISA coverage, each with its own acceptance tests.

Architectural authorities for follow-up are Arm's A-profile A64 ISA and ABI
specifications; Linux's arm64 syscall/UAPI definitions and ELF ABI; QEMU's
Linux-user and TCG implementation; and EEMBC's upstream CoreMark source and run
rules. Pin the revisions used by new differential tests and benchmark artifacts.


### Sub-second fixed-workload result (batch 11)

Seven alternating control/candidate pairs used the unchanged 1000-iteration
freestanding CoreMark ELF. Control median elapsed time was **1.20 s**; the
candidate median was **0.97 s** (range **0.96–0.97 s**), about **19.2% less
elapsed time**. All ten CRC lines match and retirement/tier/memory/loop counts
are unchanged. Native bytes fell from 144,496 to 142,698. Genuine host
context checks fell from 12,733,899 to 3,879,324; they were not artificially
incremented to hide the scheduling change. These are diagnostic wall times,
not valid CoreMark scores: the upstream ten-second minimum remains unmet.

The personality now schedules at most sixteen separate <=64-instruction native
quanta per `RunQuanta` call. Original `Run` and `Step` behavior remains bounded
as before. Host accounting is aggregated without guest-register callbacks;
faults/misses/cold work stop the schedule. Single-threaded context proofs remain
valid until a slow path, while native non-executable stores cannot alter code.

Native lowering additionally fuses exit predicates, defers exit-only SSA DAGs,
reuses distinct checked read/write page facts, and emits compact amd64 compares
and mask tests. Architectural pair accesses retain the interpreter handler but
use one checked backing page for same-page reads/writes. A pair store still
allocates exactly one generation before either value is written; cross-page
accesses keep complete prevalidation. This is implementation atomicity, not a
claim about real AArch64 pair atomicity. Existing resource limits are unchanged.
See `aarch64-validation.md` and `aarch64-coremark-results.json` for all paired
measurements, discarded experiments and correctness evidence.
