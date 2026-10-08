# AArch64 validation evidence and outstanding execution workflows

> Current benchmark summary: `aarch64-coremark-score.md`. Dated sections below
> retain development history; older results and proposals are not current claims.
> Historical raw trial/profile filenames refer to local artifacts not included
> in this PR. Current score evidence is checked in alongside the summary.

## Implemented checks

The approved repository checks exercise the authoritative RFE archives, not the
ignored editing copies. The package test verifies canonical formatting, extracts
the CPU and Linux personality, resolves imports and runs all embedded tests.

Current coverage includes:

- Literal architectural vectors plus independent arbitrary-precision ADC/SBC
  carry/overflow results and all condition-code truth tables.
- Fixed-seed multi-instruction interpreter/IR/native comparisons, including
  register extension/scaling, NZCV forwarding, bitfields and variable shifts.
- End-to-end synthetic static ELF execution in all three modes. These kernels
  are correctness fixtures, never a substitute for the actual CoreMark program.
- Native modular multiplication and constant-factor strength reduction.
- Compiled terminal branches: literal target/link vectors, differential bit-test
  coverage, budget accounting, terminal-page guards and destination faults.
  A synthetic countdown ELF compiles every non-SVC instruction, including its
  loop branches; it is not a benchmark.
- Versioned page guards, cross-page code dependencies, invalidation, remapping,
  address-space identity, generation exhaustion and aliased memory.
- Original PDP-11 archive tests and shared IR regressions.
- Page-local scalar accesses and page-chunked buffers checked against independent
  byte expectations, including cross-page permissions and transactional faults.
- Executable-context mutation paths, aliases, exhausted generations, unchanged
  page revalidation and page-version-only fallback.
- Direct-cache collisions, invalidation, cache budgets and entry-PC specialization.
- Immutable prepared IR, old-state forwarding, dimension rejection and zero
  hot-path allocations with a reused bounded evaluator frame.

The compiler bundle is regenerated through the fixed repository workflow when
its pure-block record contract changes. Both stage0 backend and bundled
bootstrap builds are checked. Routine validation ends with repository preflight;
it does not replace merge-queue full validation or execution on other hosts.

## Pinned CoreMark now runs; the performance gate remains unmet

On October 7, 2026 the user provisioned the cross-compiler and a clean upstream
checkout at `d5fad6bd094899101a4e5fd53af7298160ced6ab`. The unchanged five
algorithm/framework C files and header were copied to the fixed benchmark path.
Upstream license, readme and checksum list are retained there. The original
freestanding port is preserved in `emulators/coremark-port/` with reproduction
notes. No configuration changes, downloads or arbitrary process execution were
used. The approved fixed workflows supplied the build and execution commands.

Toolchain: GCC 16.2.1 20260819 (Red Hat Cross 16.2.1-1), binutils 2.46.1-1.fc44,
QEMU 10.2.2 (qemu-10.2.2-1.fc44), Linux/amd64 host. Full compiler flags are
printed in every raw log. Host CPU characterization, pinning, ELF digest and
valid paired scoring trials have not been collected. Batch 3 includes seven
alternating-order diagnostic pairs, still below upstream's duration minimum;
these are not a publication-quality scoring comparison.

The port runs both required seed sets sequentially using unmodified upstream
main, volatile seeds, the standard 2000-byte total dataset, stack allocation,
no libc or FP, and real CLOCK_MONOTONIC microsecond timing. All units use the
same fixed flags. The port uses only write, clock_gettime and exit. Upstream
HAS_FLOAT=0 output truncates seconds and rates; use `Total ticks` for precise
elapsed time. Upstream main returns zero even on validation errors, so exit
status alone is not accepted as evidence.

### Compatibility closure

The initial interpreter run stopped on GCC's PACIASP prologue. The guest CPU
advertises no PAC/BTI support. Explicit backward-compatible PACIASP/PACIBSP,
AUTIASP/AUTIBSP and BTI HINT variants now implement the corresponding base-CPU
NOP behavior; non-HINT authentication instructions remain unsupported. No build
flags were changed to suppress these instructions. Widening multiply-add/subtract
and signed/unsigned high-half multiply were also implemented in interpreter,
IR and native tiers. Independent arbitrary-precision multiplication regressions
cover signed bounds, limb carries, operand/destination aliases and zero-register
roles. Unsupported and malformed encodings remain rejected.

Both seed sets pass their expected list/matrix/state CRCs in QEMU, interpreter,
IR and native modes at 1 and 100 iterations. At 100 iterations all four agree on
final CRCs 0x988c (performance) and 0x844d (validation). The known CRC tuples are:

| Seed set | Seed | List | Matrix | State |
| --- | --- | --- | --- | --- |
| Performance | e9f5 | e714 | 1fd7 | 8e3a |
| Validation | 18f2 | e3c1 | 0747 | 8d84 |

### Measurements and first measured optimization

The initial 100-iteration native run spent 4.50 seconds whole-process versus
4.07 interpreter, 7.75 IR and 0.03 QEMU. It performed 13,709,698 translation
instruction-word reads. Cache entries now also retain interpreter-only words
(memory, division, traps, unsupported instructions), guarded by the same code
versions/execute permissions or byte checks as compiled words. Such instructions
still execute architecturally and count as interpreted; faults and invalidation
are not bypassed. Regressions cover fresh data loads, failed writeback, code
replacement in both directions, unsupported instructions and revoked permission.

After this change, the same 100-iteration ELF took 2.58 seconds in native mode,
with 9,909 translation instruction-word reads. These are single before/after
observations, not a statistical speedup claim. Guard checks increased because
interpreter-only entries now have explicit guards.

The subsequent identical-ELF 1000-iteration run produced:

| Engine | Performance timed interval | Validation timed interval | Whole process |
| --- | ---: | ---: | ---: |
| Native, guarded cache | 12.852696 s | 12.755269 s | 25.61 s |
| QEMU | 0.133174 s | 0.129913 s | 0.27 s |

Native printed upstream's successful validation message for both seed sets,
with final CRCs 0xd340 and 0x26c2 matching QEMU. Both native intervals exceeded
ten seconds. **QEMU's intervals did not meet the ten-second minimum**, so this
is not a valid paired scoring result and does not establish a reportable QEMU
score or speedup. QEMU is nevertheless plainly far ahead in these diagnostics.
The shorter runs print upstream's expected minimum-duration error; none had
algorithm CRC errors.

Raw stdout/stderr, iteration counts, engine labels and truncation/success flags
are retained in `aarch64-coremark-results.json`. Every comparison at a given
iteration count ran one unchanged guest ELF; rebuilding occurred only when
changing that count. Guest retired counts can differ slightly when timing
changes which reporting branches execute.

The performance goal remains unmet. At the observed gap, simply increasing the
shared workload enough for QEMU's ten-second minimum would exceed the approved
180-second native deadline. Do not weaken that limit or substitute a smaller
native workload. Checked scalar native memory and bounded native linking are now implemented
(batch 3 below). Better instruction selection, broader native memory forms and
optimizing regions remain implementation work.

### Performance optimization batch 1

The 1000-iteration guest ELF was not rebuilt during this batch. Successive fresh
native processes measured 20.91 s after page-local scalar accesses/chunked
buffers, 17.14 s after executable-context guards, and 14.31 s after direct
lookup, PC specialization and prepared IR. All algorithm CRCs still matched.

Three final diagnostic pairs, in native/QEMU, QEMU/native, native/QEMU order,
produced native whole-process times 14.19, 14.28, 14.15 s (median 14.19 s), while
QEMU remained 0.27 s in each pair. Native performance/validation timed intervals
were approximately 7.1 s each. Neither engine met the ten-second requirement at
this iteration count after optimization, so these pairs are diagnostic evidence,
not a published score or the seven-pair performance acceptance gate. The earlier
25.61 s native result is a historical baseline, not an interleaved control.

Native code size fell from 167,063 to 132,102 bytes. Page-version checks fell
from roughly 326 million to 2,325, with roughly 326 million constant-time context
checks instead. Guard semantics are preserved by a new explicit memory contract,
not by ignoring code writes or permissions. An unrelated executable context
change still forces page validation; only unchanged contexts reuse validation.
The new memory paths use one map lookup for the common scalar page and copy
buffers page by page. Cross-page writes still validate all pages before committing.

The optimized IR-only mode completed in 20.20 s, with matching CRCs, both timed
intervals just above ten seconds, and no native execution. These timings isolate
only whole-engine behavior, not detailed hotspot attribution.

**At the end of batch 1, native guest-memory emission and native linking were absent.**
The batch removes overhead around architectural memory execution and pure native
blocks; it is not the full memory JIT. Native-arena locking, native-entry checks,
resource gates and instruction retirement limits remain unchanged.


### Profile-guided optimization batch 2

The approved fixed profiling workflow samples only the same native runner and
unchanged 1000-iteration guest. Its baseline main PMU group had 1454 samples,
with hotspots in native entry maps/locking, guarded block dispatch and redundant
pure/extra decoding of cached memory instructions. Hybrid-host secondary PMU
groups with only a few samples are not interpreted as hotspot evidence. These
short profiles guide development, not precise attribution or acceptance scores.
The raw reports and profile output are retained alongside the result JSON.

Implemented changes:

- Cached scalar/pair words dispatch directly to architectural memory handlers;
  dynamic addresses, encoding rejection, permissions and precise writeback are
  still checked. Division, literals, traps and unsupported words keep Execute.
- Native entry metadata uses a bounded dense offset-indexed table instead of two
  maps. Exact state size is checked under arena serialization, together with
  alignment, installed-entry and closed/broken-arena checks.
- A shared 64-slot page lookup cache retains live page ownership across aliases.
  Permissions are rechecked; unmap clears cached ownership before remap.
- A bounded 64-instruction host dispatch quantum amortizes the arena lock across
  hot pure blocks and cached architectural memory instructions. Every block is
  still code-validated, every native entry is checked, and faults/traps stop
  with exact PC and retirement. Cold compilation never runs under that lock.
- Cold decoding/allocation is separated from hot guarded lookup. Direct cache
  hits avoid rewriting their GC-tracked slot, and fallback reuses a lookup
  already checked at the same PC.

Successive diagnostic whole-process native times were 8.64 s after cached
memory dispatch/dense metadata, 7.90 s after shared page lookup/guard ordering,
7.27 s after bounded batching, and 6.82 s after hot/cold dispatch separation.
The guest ELF was not rebuilt. Algorithm CRCs and final CRCs d340/26c2 match;
the retirement count remained 591,617,266 throughout this batch.

Three final alternating-order pairs produced native times 6.81, 6.82, 6.82 s
(median 6.82 s), versus QEMU 0.32, 0.29, 0.27 s whole-process. QEMU's timed
performance/validation guest intervals remained approximately 0.13 s each;
native intervals were approximately 3.41 s each. Whole-process QEMU results
include visibly variable startup/host time, so do not infer guest throughput
from those overheads. Compared with the prior batch's 14.19 s median, native
runtime approximately halved. That baseline is historical, not an interleaved
control. Both engines still fail upstream's ten-second duration requirement:
these are **diagnostics, not valid CoreMark scores or a QEMU win**.

New regressions cover native entry holes/alignment, exact state shape, arena
exhaustion/closure, bounded batch limits, failed-call accounting and concurrent
Close serialization; cold/cached memory execution vs architectural execution;
aliased page-cache collisions, protection/unmap/remap and transactional faults;
and native dispatch quanta with precise budgets, faults and self-modifying code
that replaces a hot branch with an unsupported instruction or SVC.

Native code remains 132,102 bytes. Final runs performed 2,325 page-version checks
and 331,627,287 context queries, with zero invalidations/compile failures for this
fixed read-only-code guest. **At the end of batch 2, native memory emission and
native linking were unimplemented.** Batch 3 implements those baseline paths;
the optimizing-region engine and required QEMU win remain unimplemented.


### Effect-aware native scalar memory and native linking (batch 3)

This batch implements native guest-memory execution, rather than only making
architectural memory dispatch cheaper. Mixed pure/scalar blocks use ordered
memory effects, explicit architectural checkpoints and precise slow exits.
Byte/halfword/word/doubleword loads and stores, signed loads, supported immediate
and extended register addresses, SP guards, ZR operands and pre/post-index
writeback preserve the reference semantics. Pairs and literals remain slow paths.
The portable evaluator remains pure; effect blocks stay architectural until
native promotion. No load/store is CSE'd or reordered past a checkpoint.

The Linux memory provides a shared 64-slot, full-page-tagged native table with
GC-visible pointers. Generated accesses validate the 48-bit range, page boundary,
permissions and ownership. Native stores require non-executable writable pages,
allocate a non-reused clock generation and update the actual page stamp. Misses,
executable stores, crossing accesses and failed guards return before the access;
the architectural slow path executes only that instruction, not its prefix.
Protection/unmap/remap invalidate descriptors across memory aliases.

A native dispatcher now traverses published hot blocks within one arena lock
and one unchanged executable context, with a maximum of 64 retired instructions.
Each transition checks PC/tag, whole-block budget, installed aligned offset and
exact state shape. The live arena view is cleared on return; dispatcher entries
are excluded as leaf targets. Published offsets are cleared on executable-context
or native-arena-owner changes. No compilation, Go callback, executable-page store
or syscall occurs inside the native loop. Missing successors return to guarded
host dispatch. Custom memories without this contract retain the old fallback.

The unchanged 1000-iteration guest first measured 5.35 s with native scalar
memory, then 4.34 s with native linking. Both seed sets and final CRCs d340/26c2
match. Seven alternating-order pairs (four native/QEMU, three QEMU/native) then
measured:

| Engine/build | Whole-process elapsed time | Median | Timed intervals per seed set |
| --- | --- | ---: | --- |
| Batch 2 historical native | 6.81, 6.82, 6.82 s | 6.82 s | about 3.41 s |
| Batch 3 native diagnostic pairs | 4.34, 4.34, 4.34, 4.34, 4.35, 4.35, 4.34 s | 4.34 s | about 2.17 s |
| Paired QEMU | 0.28, 0.27, 0.27, 0.27, 0.27, 0.27, 0.27 s | 0.27 s | about 0.13 s |

An additional progress-hardening change rejects impossible partial records
before accounting or prefix indexing. Its final rebuilt-runner confirmation
measured 4.39 s (2.196208/2.194254 s guest intervals) with the same CRCs and tier
counts. The seven pairs and profile precede that final defensive change, rather
than being mislabelled as trials of identical final native bytes. The guest ELF
was never rebuilt during this batch. All raw outputs, order labels, success and
truncation flags are retained in `aarch64-coremark-results.json`.

The seven-pair observed native range was 4.34–4.35 s at the timer's 0.01 s
whole-process resolution; this narrow range is not a confidence interval or a
claim of controlled host conditions. Relative to the historical 6.82 s median,
the paired median is roughly 36% less elapsed time. That historical baseline
was not an interleaved control. QEMU remains roughly sixteen times faster in
these whole-process diagnostics. **Neither engine meets the ten-second minimum;
these are not valid CoreMark scores or a performance acceptance pass.**

Final native counters: 591,617,266 total retired; 1,755,417 interpreted;
66,683 portable IR; 589,795,166 native, including 135,161,653 native scalar memory
instructions. 589,793,523 instructions retired inside the native dispatcher.
There were no memory slow exits, invalidations or compile failures for this
fixed guest. Native code is 262,575 bytes including the hardened dispatcher.
Context checks fell from 149,028,823 after memory emission to 12,555,223 with
linking; page-version checks remain 2,327. These counts are not claims about
fault frequency or invalidation cost on other programs.

The memory-only profile had 532 main-group samples and zero lost samples; its
largest named host costs were guarded block lookup, batch selection, arena
entry calls and memory accounting. The linked profile had 441 main-group samples
(and a separate 3-sample PMU group that is not interpreted), zero lost samples,
and much less named host dispatch overhead. Many samples are now spread among
anonymous generated code addresses. This points toward generated instruction
count/SSA spills and native dispatch checks as further profiling targets, but
these short, unsymbolized samples do not isolate their exact proportions.
Raw reports are `aarch64-coremark-profile-memory.txt` and
`aarch64-coremark-profile-linking.txt`.

New checks cover all native access widths, permission masks, unsigned extension,
page/range misses, generation exhaustion, non-removable ZR loads, checkpoint
ordering and load/store/load non-CSE. Embedded architectural differential tests
cover scalar forms, signed/register addresses, writeback, SP and overlapping
registers. Linking regressions cover odd/exact budgets, invalid entry offsets,
shape/dispatcher rejection, ownership swaps between arenas, address-space
replacement, executable mutations through aliases, permission revocation,
unmap/remap, committed-store partial faults and resumption without replay.
Invalid partial progress, closed arenas and unbounded call requests are rejected.

Both compiler builds, focused runtime tests, all embedded emulator package
tests, and the required repository preflight pass with resource gates unchanged.
Actual native execution was measured on Linux/amd64 only; the AArch64 emitter
path is not independently executed on an AArch64 host by these workflows.
The QEMU win and optimizing-region milestone remain open. Next implementation
work is better native instruction selection/register residency and broader
native memory coverage, not further claims based on shortened scores.


### Register-resident dispatch and compact native emission (batch 4)

Implemented a register-resident native dispatcher on amd64 and arm64: state,
context, remaining budget, link descriptor and block length stay in preserved
registers across leaf calls. The quantum remains **64 instructions**, with the
same PC/tag, entry, state-shape, ownership, whole-block-budget and partial-progress
checks. No resource gate or guest memory limit was relaxed.

Native SSA temporaries now use bounded live intervals, constant rematerialization
and spill fallback. Pure blocks have six allocatable registers; amd64 memory
blocks reserve guard scratch and use four. Guest architectural registers are
still committed at block boundaries; this is not cross-block guest-register
residency or an optimizing-region engine. Compact memory guards preserve fault
ordering, permissions, non-executable store requirements and page-generation
updates. Native immediate operations, variable shifts/rotates and conditional
moves replace expanded IR sequences. Select alternatives remain already-computed
values: an unselected load still faults. Immutable link descriptors avoid repeated
prefix copying, and fresh block guards supply the executable context without a
redundant host query.

Opt-in native profiling now emits exact address/length names for generated blocks
and the dispatcher. On Linux, `-stats` enables an append-only `/tmp/perf-PID.map`
created exclusively with mode 0600; it is retained for later perf reports.
Profiling errors are nonfatal and counted. The initial symbolized profile placed
about 39% of main-group samples in the dispatcher. The final profile has 236
main-group samples, zero lost samples, and about 23% in that dispatcher; its
separate three-sample PMU group is too small to interpret. Sampling proportions
are diagnostic, not precise attribution of each optimization's benefit.

Seven final alternating-order pairs used the **unchanged 1000-iteration ELF**:

| Engine/build | Whole-process elapsed time | Median |
| --- | --- | ---: |
| Batch 3 hardened native, historical confirmation | 4.39 s | — |
| Batch 4 final native | 2.32, 2.30, 2.30, 2.29, 2.31, 2.29, 2.29 s | 2.30 s |
| Paired QEMU | 0.27 s in all seven trials | 0.27 s |

Compared with the historical hardened confirmation, the median is approximately
48% less elapsed time (1.91x faster). That baseline was not an interleaved control.
The observed 2.29–2.32 s range is not a confidence interval. Both engines matched
all expected performance and validation CRCs, including final d340/26c2, but both
fail upstream's ten-second duration requirement. **These remain diagnostics, not
valid CoreMark scores, a QEMU win, or a performance acceptance pass.**

Final generated code is **155,564 bytes**, down from 262,575 (about 41% smaller).
Final counters: 591,617,306 retired; 1,755,421 interpreted; 66,689 portable IR;
589,795,196 native; 135,161,661 native scalar memory; 589,793,553 linked;
2,327 page-version checks; 12,555,233 context checks. Compilation failures,
memory slow exits, invalidations and profile errors are zero for this guest.
The additional 40 retired instructions relative to batch 3 come from the guest's
elapsed-time reporting branch when its intervals change from roughly two seconds
to one, not from a changed guest ELF or relaxed instruction ceiling.

Both compiler builds, focused native/IR checks, all emulator archive tests and
repository preflight pass. Added independent tests cover register pressure and
state versions, variable shift/rotate widths and masked counts, signed-32 immediate
boundaries, select truthiness/third-operand validation/remapping, unselected-load
faults, and symbol record bounds. Actual execution is verified on Linux/amd64;
the arm64 emitter path has not been executed on an arm64 host in these workflows.
Raw outputs and order labels are in `aarch64-coremark-results.json`; symbolized
reports are `aarch64-coremark-profile-symbols.txt`,
`aarch64-coremark-profile-registers.txt` and `aarch64-coremark-profile-final.txt`.
Cross-block guest-register residency, optimizing regions and broader native
memory coverage remain follow-up work.


### Compact flags, direct predicates and cold exit metadata (batch 5)

This performance-only batch adds width-explicit arithmetic/logical NZCV IR,
compact native flag capture and direct condition predicates. A branch/select
using a just-computed arithmetic condition no longer has to pack NZCV and
immediately unpack it. Architectural flag stores remain live at exits; ADC/SBC's
carry-in-aware lowering is unchanged. Native leaf blocks keep their architectural
state base in a dedicated caller-saved register, removing repeated stack reloads
of that pointer. The amd64 SSA pools are now five registers for pure blocks and
three for memory blocks; arm64 retains six SSA registers plus a separate state
base. This is pointer residency within a leaf, not cross-block guest-register
residency.

Memory checkpoints still commit dirty architectural state in the original order.
Only native progress metadata is deferred to the successful return or an
access-specific slow-exit stub. Fault addresses are reconstructed from the
original live SSA operand in that stub, before the access result can overwrite
its register. Fast accesses no longer save an address to the stack or update
retirement metadata at every checkpoint. Permission, ownership, range, page
boundary and generation checks are unchanged; faults retain the exact committed
prefix without replaying stores. The 16-instruction block ceiling, 64-instruction
quantum, cache, native-arena and guest-memory limits are unchanged.

An exploratory pure forward-branch tracing change passed the narrowed effect
boundary checks but showed no measured improvement (1.92 s, versus 1.92 s before
tracing) and increased code from 128,385 to 134,800 bytes. It was removed rather
than retained as speculative complexity. No region engine is claimed.

Final seven alternating-order pairs, using the unchanged 1000-iteration ELF:

| Engine/build | Whole-process elapsed time | Median |
| --- | --- | ---: |
| Batch 4 native, historical paired median | 2.30 s | 2.30 s |
| Batch 5 final native | 1.80, 1.79, 1.80, 1.83, 1.80, 1.80, 1.80 s | 1.80 s |
| Paired QEMU | 0.28, 0.27, 0.28, 0.27, 0.27, 0.27, 0.27 s | 0.27 s |

The native median is about 22% less elapsed time than the previous batch, or
1.28x faster. The previous native build was not an interleaved control, and the
observed range is not a confidence interval. Code is 116,465 bytes, about 25%
smaller than batch 4's 155,564. All expected seed/list/matrix/state/final CRCs
match. These runs are still below upstream's ten-second minimum, not valid
CoreMark scores or a QEMU win.

Final counters: 591,616,306 retired; 1,755,313 interpreted; 66,517 portable IR;
589,794,476 native; 135,161,459 native scalar memory; 589,792,833 linked;
2,315 blocks; 319 promotions; 2,317 page-version checks; 12,555,020 context checks.
Compile failures, memory slow exits, invalidations and profiling errors are zero
for this guest. Relative to batch 4, 1,000 fewer instructions retire because the
sub-second intervals take a different elapsed-time/rate reporting path in the
unchanged guest. Benchmark iterations and computation were not reduced.

The final profile has 184 main-group samples, zero lost samples and approximately
32% in the native dispatcher, with a separate three-sample group too small to
interpret. Its larger relative share after shrinking block work does not establish
an increase in absolute dispatcher time. Reports are
`aarch64-coremark-profile-flags.txt` and `aarch64-coremark-profile-exits.txt`;
raw outputs, including the rejected experiment and every paired trial, are in
`aarch64-coremark-results.json`.

Focused flag checks use independent arbitrary-precision overflow expectations,
all sixteen condition truth values, both widths and live-register pressure.
A targeted slow-exit check keeps an old spilled address after its architectural
slot changes. Existing native-memory, fault, linking and emulator archive checks
pass. Both compiler builds and required repository preflight pass; preflight
completed in 28 seconds under the unchanged 60-second budget. Broader compatibility
work, full validation and actual arm64-host execution remain separate work.


### Bounded self-loop regions and cold dispatcher accounting (batch 6)

The linked Run path can now compile two iterations of an already-promoted,
direct self-loop into one guarded native entry. B, B.cond, CBZ/CBNZ and TBZ/TBNZ
back edges qualify; calls and indirect branches do not. Expansion is admitted
only when the doubled static instruction count remains within the unchanged
16-instruction ceiling. Original instruction words, dependency pages and the
single-block entry remain intact. Step still executes one original block, and
Run uses that entry when a region does not fit the remaining budget. This is
bounded loop expansion, not a general multi-block CFG/loop-phi engine.

An effect-only RegionGuard follows the first iteration's architectural commit.
It tests the actual branch successor; a failed condition returns status 4 with
that completed prefix, not a fault or an instruction for architectural replay.
The dispatcher still rejects partial progress at or beyond the region's declared
count before indexing the memory-prefix table. Second-iteration memory accesses
carry their exact accumulated retirement and original fault PC. No access,
permission, generation or executable-context check is removed. Regions use only
the original block's decoded words/pages; arena, cache, quantum and guest-memory
limits are unchanged. A constant-select equality simplification avoids selecting
an exit PC only to compare it back to the loop entry. Successful dispatcher
accounting no longer reloads an already-checked zero status; cold exits account
their prefix once and return immediately.

A one-entry admission memo, a checked self-edge fast path, and return-only total
accounting were also tried. They did not establish an improvement and were
removed. Raw trial outputs and a trial profile are retained as evidence, not as
features or claimed gains.

The unchanged 1000-iteration ELF produced these whole-process timings:

| Engine/build | Elapsed time | Median |
| --- | --- | ---: |
| Same-session pre-change native, three confirmations | 1.81, 1.80, 1.79 s | 1.80 s |
| Batch 6 final native, seven trials | 1.75, 1.76, 1.79, 1.76, 1.76, 1.76, 1.75 s | 1.76 s |
| QEMU, paired after each final native trial | 0.27, 0.27, 0.27, 0.31, 0.27, 0.28, 0.27 s | 0.27 s |

The observed median improvement is modest: about **2.2% less elapsed time**, or
1.023x faster. Baseline confirmations preceded the final pairs, rather than
being an interleaved old/new control; this is not a confidence interval. All
expected performance and validation CRCs match. Both engines still fail the
upstream ten-second duration requirement: these are diagnostic timings, not
valid CoreMark scores, a QEMU win or a performance acceptance pass.

Final generated code is 128,204 bytes, about 10% larger than batch 5's 116,465.
The unchanged retired/tier totals are 591,616,306 retired, 1,755,313 interpreted,
66,517 IR, 589,794,476 native, 589,792,833 linked and 135,161,459 native scalar
memory instructions. Eight regions were compiled; 416,323 successful side exits
were taken. Code reads are 13,194, page-version checks 2,317 and executable-context
checks 12,739,271. Context checks rose by about 1.5%, since region side exits return
to Go. Compile failures, memory slow exits, invalidations and profile errors
remain zero for this benchmark.

The final profile contains 188 main-group samples, zero lost samples and about
29% in the dispatcher. The separate three-sample group is too small to interpret.
This short profile remains diagnostic, not precise attribution. Evidence is in
`aarch64-coremark-results.json`, `aarch64-coremark-profile-regions.txt` and
`aarch64-coremark-profile-transition-trial.txt`.

Focused tests cover first-iteration branch exits, short/exact instruction budgets,
unchanged Step behavior, the static expansion ceiling, a second-iteration fault
without store replay, executable writes/protect/unmap/remap, pure-IR rejection
of region guards, invalid partial progress and select equality truthiness.
Both compiler builds, focused native/IR checks and embedded emulator source-package
tests pass. Required repository preflight passed in 28 seconds under the unchanged
60-second budget, including generated-source checks and test compilation. Actual
native execution is still verified only on Linux/amd64; arm64-host execution,
full compatibility validation and broader native coverage remain separate work.


### Resident loop traces, native exit maps and rich integer IR (batch 7)

This supersedes batch 6's two-iteration expansion. One static iteration now has
native loop-carried phis and compile-time exit maps. Ordinary arithmetic,
selects and shifts emit directly in allocated registers rather than repeatedly
loading AX/X0 and copying the answer back. Parallel phi copies use direct
register moves; cycles use a private temporary and pressure uses private spills.
Architecture-constrained multiply/flags operations retain reserved scratch.

Hot direct self-loops and cyclic traces through already-validated native blocks
keep guest values resident across iterations and internal guest branches.
The entire static trace still fits the **16-instruction ceiling**, and execution
still yields at the **64-instruction native quantum**. Trace construction does
not discover new code pages or relax cache/arena limits. Every constituent code
page is added to the entry's dependency set. Step retains its original leaf;
short Run budgets select that leaf instead of the longer trace.

Architectural StoreState records inside a region update compile-time state maps,
not the hot architectural array. Generated cold native stubs reconstruct the
exact completed prefix from registers, spills and constants on an exit. Internal
branch side exits account that prefix and continue the native dispatcher; they
are not Go callbacks for register assignments or guest-state reconstruction.
Memory/trap exits return to architectural execution before the pending access.
Published normal successors also remain in native dispatch when available.

A private one-page translation fact survives within the locked native loop call.
Hits reuse the full guest-page tag, host backing pointer, permissions and page
epoch pointer. Every access still checks address range, width/page boundary and
required permissions; every store still rejects executable mappings and checks
and updates the live epoch/clock without wrapping. The fact is reset on entry
and never survives Go mapping/protect/unmap operations or a later native call.

High-half signed/unsigned multiply, CLZ/CLS, byte/bit reversal and carry arithmetic
with explicit three-operand carry/NZCV now survive as rich IR records until host
selection. amd64 uses baseline MUL/IMUL, BSR with an explicit zero case, BSWAP,
bit-swap stages and ADC/SBB with carry loaded from SSA. It assumes no LZCNT/BMI
extensions. arm64 emits corresponding native integer instructions, with an
explicit NZCV carry input. Both portable and native results are checked against
arbitrary-precision arithmetic and width/zero/signed-boundary vectors. A real
Renvo-compiled portable consumer prints PASS; the reference evaluator does not
require unavailable math/bits APIs from Renvo's current small standard library.

The unchanged 1000-iteration ELF produced these whole-process diagnostics:

| Engine/build | Seven elapsed times | Median |
| --- | --- | ---: |
| Batch 6 native (previous recorded baseline) | 1.75, 1.76, 1.79, 1.76, 1.76, 1.76, 1.75 s | 1.76 s |
| Batch 7 native | 1.41, 1.41, 1.41, 1.45, 1.42, 1.41, 1.41 s | 1.41 s |
| QEMU, after each batch 7 native trial | 0.27, 0.27, 0.27, 0.27, 0.27, 0.27, 0.27 s | 0.27 s |

This is **19.9% less median elapsed time**, or about **1.25x faster than batch 6**.
The old/new measurements are not interleaved controls or a confidence interval.
The paired runner preceded a portable-reference-library compatibility fix;
native emitted bytes/counters remained identical, and a confirmation afterwards
was 1.40 s. All expected CRCs match. These shortened runs still fail upstream's
ten-second scoring requirement; **QEMU is still about 5.2x faster**, and neither
a valid score nor the performance acceptance gate is claimed.

Final code occupies **142,141 bytes**, up from 128,204. There are 30 bounded
regions, 42,737,643 native loop iterations and 8,097,762 successful exits.
Retirement/tier/memory/link totals remain exactly unchanged: 591,616,306 retired,
1,755,313 interpreted, 66,517 IR, 589,794,476 native, 589,792,833 linked and
135,161,459 native scalar memory instructions. Code reads/page-version checks
remain 13,194/2,317; context checks are 12,733,899. Compile failures, native
memory slow exits, invalidations and profile errors are zero on this guest.

The diagnostic profile has 146 main-group samples, two separate tiny-group
samples and zero lost samples. The dispatcher is 19.66% of the main group,
compared with about 29% in batch 6; the arena CallLinked wrapper is 10.94%.
Sampling is too short for precise attribution. Full evidence is retained in
`aarch64-coremark-results.json` and `aarch64-coremark-profile-resident-loops.txt`.

Focused tests additionally exercise register/spill phi cycles, uneven and exact
budgets, native branch-side-exit chaining, third-operand carry liveness,
second-iteration faults without replay, read-only/executable cached-store
rejection, backing-page replacement across calls, clock exhaustion and all
constituent-page write/protect/unmap/remap invalidation. Source-package tests,
compiler builds and required preflight pass with resource gates intact. Final
preflight completed in 29 seconds under its unchanged 60-second budget. A
zero-progress branch side exit is rejected by both the runtime and raw native
emitter, preventing a dispatcher cycle that cannot consume its budget.
Native execution is verified on Linux/amd64 only; arm64-host execution remains
unverified. These are bounded cyclic traces, not a general CFG optimizer:
non-trace transitions still materialize state, and cold compilation, mapping,
traps and bounded-quantum scheduling still run in Go. No claim of eliminating
all Go transitions or arbitrary-control-flow state residency is made.


### Invariant accesses, cold state and immutable admission (batch 8)

The sub-second whole-process target remains **unmet**. This iteration keeps the
same 1000-iteration CoreMark ELF, guest flags, seed sets, thresholds, 16-static-
instruction trace limit and 64-dynamic-instruction native quantum. No larger
block/quantum, Go register-assignment callback or guest-specific shortcut is used.

Changes:

- A specialized arena dispatcher wrapper keeps one serialized cleanup for the
  borrowed code view and lock, validates dispatcher family/state shape, and
  roots the state/context/private stack/typed metadata throughout the call.
- Loop-invariant SSA addresses get an exact-access translation fact after all
  range, boundary, tag, pointer and permission guards succeed. The fact lasts
  only for one serialized single-threaded loop call; it never caches a loaded
  value. Non-invariant addresses retain their per-access guards. Stores still
  check clock overflow on every iteration and update actual epoch/data.
- Exit-only arithmetic/logical NZCV and selects are computed in cold native
  restoration stubs, not every iteration. Their operands stay live through the
  relevant exit maps. Any ordinary/effect use or loop-phi use keeps a value hot.
  Unchanged architectural input slots need no redundant restoration stores.
- Cold link publication installs an immutable pointer-free arena admission key.
  Every native selection still checks PC/tag, budget, offset alignment/index,
  requested body count and exact state shape. The private key proves the entry
  was installed with an admissible family. Public descriptors cannot forge it;
  unadmitted entries retain the original full checks. Four-byte metadata records
  replace two-byte records: at the default 8 MiB code ceiling their logical
  payload is at most 2 MiB instead of 1 MiB, excluding slice-capacity overhead.
- Dispatcher ABI preparation, unchanged region-attempt work and prefix copies
  move off repeated host calls. Owner/executable-context guards and all progress
  validation remain. Guest values and exit reconstruction stay in native code.

Final alternating-order pairs on the same Linux/amd64 host:

| Engine/build | Seven whole-process elapsed times | Median |
| --- | --- | ---: |
| Batch 7 native, previously recorded | 1.41, 1.41, 1.41, 1.45, 1.42, 1.41, 1.41 s | 1.41 s |
| Batch 8 final native | 1.35, 1.34, 1.34, 1.34, 1.31, 1.34, 1.33 s | 1.34 s |
| QEMU in the final pairs | 0.28, 0.27, 0.27, 0.27, 0.27, 0.27, 0.27 s | 0.27 s |

The recorded median is **5.0% lower** than batch 7, about **1.05x faster**.
The old/new builds were not interleaved controls; this is not a confidence
interval. This turn's one baseline trial was 1.47 s, and incremental trials
reached 1.29 s; neither single observation replaces the final repeated median.
An earlier wide-admission-record series had a 1.34 s median as well. All raw
stage/pair logs are retained, not only the fastest observation.

Both expected ten-value CRC tuples match in every stage, paired run and profile.
All retirement, tier, linked, memory and region counters remain identical to
batch 7: 591,616,306 retired; 589,794,476 native; 589,792,833 linked;
135,161,459 native memory; 30 regions; 42,737,643 loop iterations; 8,097,762
region exits; 13,194 translation code reads; 2,317 version checks; 12,733,899
context checks. Compile failures, native memory exits, invalidations and profile
errors are zero. Final generated code uses **141,871 bytes** (previously 142,141).

These are shortened diagnostics, not valid upstream CoreMark scores: both seed
runs still print the expected ten-second minimum-duration error. QEMU remains
about **5.0x faster** on these diagnostics. Neither a scoring win, the QEMU
performance gate nor a sub-second whole-process result is claimed.

The final profile has 142 main-group samples and zero lost samples. About 23.02%
is in the native dispatcher, 6.90% in the arena CallLinked wrapper, 5.17% in
Native.CallLinked, 7.10% in guarded block lookup and 3.08% in runLinked. Short
sampling is noisy; these are directions for further work, not precise causal
speedup attribution. Reports and CRC-validated profile logs are retained in
`aarch64-coremark-profile-compact-admission-final.txt`, the two trial profiles,
and `aarch64-coremark-results.json`. Dispatch/quantum boundaries remain the next
profiling target; the budgets were not enlarged to hide their cost.

New regressions exercise cold flags/selects with register pressure/private
spills, normal exits and a second cached store at an exhausted clock, precise
prefix accounting without replay, old-NZCV loop phis, unchanged input state,
forged admitted counts, maximum 256-word/16-instruction dimensions, admission
immutability, metadata growth, loop length checks, closed arenas and wrapper
rejection/cleanup followed by valid reentry. Runtime package tests, authoritative
embedded emulator tests, both compiler builds and a real Renvo-built portable
rich-IR consumer pass. Actual native execution remains verified only on
Linux/amd64; arm64-host execution is still unverified. General CFG optimization
and state residency outside bounded traces remain unfinished.

Required repository preflight passed in **30 seconds** under the unchanged
60-second budget, including generated-source validation, tracked package tests,
bundled driver/bootstrap builds and backend/frontend test compilation.


### Shared-frame native-call ABI (batch 9)

The internal linked ABI now gives frame ownership to the dispatcher, not each
compiled block. It reserves one bounded 17,408-byte frame on the unchanged
64 KiB private stack for the quantum. A disjoint 64-byte header protects its
saved registers and entry metadata. Shared bodies inherit R11/X11 as state base
and a persistent context slot at FP-80. Pure SSA spills reserve that slot too.
All loop saves, phi temporaries, spills and translation facts are below the
header, and their computed peaks must fit the shared frame.

Pure/memory images provide a standalone compatibility entry and a private body
offset. Native dispatch checks the installed entry and exact state/body shape
before adding that arena-owned offset and calling the frame-free body. Loops
are dispatcher-only shared bodies and preserve only the owned registers actually
used, plus budget/iteration registers. The original raw framed emitters remain
available. Legacy framed entries can mix with shared bodies; dispatch explicitly
restores the state base because the legacy ABI allowed R11/X11 as scratch.
Standalone APIs retain their behavior and intentionally keep frame setup plus
one internal CALL/RET. No guest register-assignment/reconstruction callback,
larger instruction budget, larger arena or larger private stack is introduced.

`rfe-native-abi.md` documents the machine-register contract, scratch partition,
checked dual entries, stack bound and compatibility. Private metadata records
are now eight bytes, including a uint32 body offset, instead of four. Their
logical payload at the default 8 MiB code ceiling is bounded by 4 MiB instead of
2 MiB, excluding slice-capacity overhead; the exact-length table is unchanged.

Seven alternating-order **old/new native ABI** pairs were run with the identical
1000-iteration ELF. Between runs only the six explicitly snapshotted compiler,
arena and runtime files were switched; the fixed backend regeneration and runner
build workflows were checked each time. The final shared sources and runner
were restored. No benchmark guest rebuild occurred.

| Variant | Seven whole-process elapsed times | Median |
| --- | --- | ---: |
| Batch 8 framed ABI control | 1.29, 1.29, 1.33, 1.29, 1.29, 1.36, 1.33 s | 1.29 s |
| Batch 9 shared-frame ABI | 1.23, 1.24, 1.24, 1.23, 1.23, 1.23, 1.24 s | 1.23 s |

The shared ABI's recorded median is **4.7% lower**, about **1.05x faster** than
its interleaved control. This is a controlled diagnostic comparison, not a
confidence interval or valid CoreMark score. Batch 8's older separate-series
median was 1.34 s; it is not substituted for this turn's 1.29 s control to inflate
the measured improvement. The sub-second whole-process target remains unmet.

All ten expected CRC values match for both seed sets in every stage, paired run
and profile. Apart from generated-code byte count, each paired run's entire
stats line was checked equal to the baseline: 591,616,306 retired;
589,794,476 native; 589,792,833 linked; 135,161,459 native memory; 30 regions;
42,737,643 loop iterations; 8,097,762 region exits; 13,194 translation code reads;
2,317 version checks; 12,733,899 context checks. Compile failures, memory exits,
invalidations and profile errors are zero. Generated code occupies **145,396
bytes**, up from 141,871, reflecting compatibility entries and biased scratch
addressing. These short runs still print upstream's expected minimum-duration
error; no scoring win or QEMU acceptance gate is claimed.

New regressions exercise mixed shared pure/memory/loop and legacy chains,
standalone compatibility versus linking, exact/uneven budget boundaries,
800 simultaneously live arithmetic values requiring a large spill area followed
by native memory access, preservation of the persistent context slot, legacy
state-base clobbers, invalid body offsets/state dimensions, closed installation
and opaque internal entries. Existing precise-fault, cold flags/select, clock
overflow, phi-cycle, admission and code-invalidation regressions also pass.
Both compiler tools build; runtime and runimage package suites and authoritative
embedded emulator tests pass. ARM host execution remains unverified.

The final diagnostic profile has 129 main-group samples and zero lost samples.
The native dispatcher is 23.91%, runLinked 5.82%, arena CallLinked 4.88% and its
cleanup 5.69%; this sample is too short for precise causal attribution. Raw
before/after logs and the full report are retained in `aarch64-coremark-results.json`
and `aarch64-coremark-profile-shared-abi.txt`.

This is the first ABI step, not full cross-block guest-register residency.
Independent block transitions still materialize architectural values, and
progress travels through the checked context. Live guest-value transfer between
successors and a native return-progress convention remain follow-up work.
The 16-static-instruction trace ceiling and 64-dynamic-instruction quantum remain
unchanged.

Final batch 9 repository preflight passed with exit code 0, no timeout, no
stderr and no captured-output truncation under the unchanged 60-second budget.
It includes generated-source validation, tracked package tests, compiler/driver
builds and backend/frontend test compilation.


### Prepared dispatcher calls and guarded PC refinement (batch 10)

Native dispatchers are now admitted once into an opaque `LinkedCall`, including
entry family, state shape and the immutable private-stack top. Each invocation
still takes the original arena mutex and checks the owner's open/non-broken
state, exact state length and non-nil borrowed context/view. Appending code
refreshes a cached four-word arena view under the same lock; each call borrows
that current view, roots the state/context/stack/metadata graph and clears the
view before releasing serialization. Close invalidates every prepared handle.
Leaf target guards and the shared-frame v2 ABI are unchanged; the handle does
not expose a callable address or permit bypassing native target admission.

Multi-block region lowering refines its forwarded PC to the exact next target
only **after** the existing precise checkpoint and successful equality guard.
This exposes constant branch targets to the existing IR folding and exit-only
selection machinery. Failed guards preserve the alternate architectural PC;
later faults preserve the successfully taken target. No edge guard, code-page
dependency, fault checkpoint or budget is removed.

Seven alternating control/final pairs used the identical fixed 1000-iteration
CoreMark ELF. Host runtime and CPU archive sources were explicitly switched;
runner builds were checked for every run. The original shared-frame compiler
sources were unchanged in these final pairs, and the final sources/runner were
restored. The guest was not rebuilt.

| Variant | Seven whole-process elapsed times | Median |
| --- | --- | ---: |
| Batch 9 shared-frame control | 1.23, 1.23, 1.24, 1.24, 1.23, 1.23, 1.23 s | 1.23 s |
| Prepared call + guarded PC refinement | 1.20, 1.21, 1.20, 1.20, 1.21, 1.23, 1.20 s | 1.20 s |

The recorded median is **2.4% lower** (about **1.025x**). This small diagnostic
improvement is not a confidence interval or a valid upstream CoreMark score.
The sub-second whole-process target is **still unmet**, and no QEMU acceptance
win is claimed. Ten-second minimum-duration errors remain expected in both seed
runs. All ten expected CRC values match in every trial, comparison and profile.
Each final run's complete stats line, except generated-code bytes, was checked
against the baseline: 591,616,306 retired; 589,794,476 native;
589,792,833 linked; 135,161,459 native memory; 30 regions; 42,737,643 loop
iterations; 8,097,762 region exits; 13,194 translation reads; 2,317 version
checks; 12,733,899 context checks. Compile failures, memory exits, invalidations
and profile errors remain zero. Generated code occupies **144,496 bytes**, down
from 145,396.

Several alternatives were tried and deliberately **not retained**. Exit-only
cumulative retirement accounting, a register progress/status return convention,
and a dispatcher memory-counter register did not establish a timing win.
The register-return trial initially failed the existing invariant-access tests
because removing status initialization also removed cache-validity zeroing;
that issue was diagnosed and corrected before further trials, then the entire
return-ABI experiment was dropped. A rotating five-round comparison recorded:

| Candidate | Five elapsed times | Median |
| --- | --- | ---: |
| Shared-frame control | 1.24, 1.24, 1.23, 1.23, 1.23 s | 1.23 s |
| Prepared call/view only | 1.22, 1.22, 1.27, 1.26, 1.22 s | 1.22 s |
| Register returns + prepared call/view | 1.24, 1.24, 1.24, 1.20, 1.22 s | 1.24 s |

A terminal select/equality projection experiment was also removed: the IR
builder already performs that transformation, so it changed no benchmark code.
Raw trial and controlled logs are retained in `aarch64-coremark-results.json`;
no fastest-single-run result is substituted for the controlled medians.

New regressions cover prepared-call rejection, zero handles, state/context/view
bounds, metadata growth after preparation, concurrent callers sharing the private
stack, cleanup/reentry and closed handles. An authoritative embedded trace test
covers both a failed internal guard and a memory fault after a successful guard.
Existing multi-block budgets, constituent-page invalidation, precise faults and
shared/legacy interoperability tests also pass. Runtime/runimage suites and
embedded emulator tests pass; both fixed compiler tools build. Final repository
preflight passed in **29 seconds** under the unchanged **60-second** budget,
with no timeout, stderr or captured-output truncation.

The final profile has 124 cpu_core samples plus one cpu_atom sample, with zero
lost samples. Main-group dispatcher share is 24.27%, prepared call 9.22%,
runLinked 7.52%, block lookup 5.21%, Native.CallLinked 3.39% and prepared-call
cleanup 3.39%. This short sample cannot support precise causal attribution;
dispatch and Go quantum boundaries remain the primary follow-up targets. The
full report is `aarch64-coremark-profile-prepared-refinement.txt`.

The static 16-instruction ceiling, dynamic 64-instruction quantum, 64 KiB private
stack, default block/code limits and guest RAM limit are unchanged. Actual native
execution remains Linux/amd64-only evidence. Cross-block live guest-value
transfer, native progress return conventions and general CFG optimization remain
unfinished. The next reduction must address these larger costs rather than
increase quantum budgets or select a noisy minimum.


### Sub-second fixed workload: aggregate scheduling and compact selection (batch 11)

**The requested sub-second diagnostic target is reached.** Seven alternating
control/candidate pairs used the same fixed 1000-iteration ELF; no benchmark
source, seed, iteration count, flags, guest RAM or native resource gate changed.
Every generated-source regeneration and fixed runner build was checked before
measurement. Measurements are elapsed seconds for the complete performance plus
validation guest process, including startup, translation and shutdown:

| Pair | Batch 10 control | Batch 11 candidate |
| --- | ---: | ---: |
| 1 | 1.20 | 0.97 |
| 2 | 1.20 | 0.96 |
| 3 | 1.20 | 0.97 |
| 4 | 1.20 | 0.96 |
| 5 | 1.20 | 0.96 |
| 6 | 1.20 | 0.97 |
| 7 | 1.22 | 0.97 |
| **Median** | **1.20** | **0.97** |

This is **19.2% less elapsed time**, about **1.24x** throughput for this fixed
workload, not a claim of QEMU superiority or a valid CoreMark score. All seven
candidate runs are below one second, rather than selecting one fastest run.
The guest still reports its expected minimum-duration errors; only those
errors are accepted by the verifier. All ten CRC lines match every run.

Retired instructions remain 591,616,306; interpreted 1,755,313; IR 66,517;
native 589,794,476; linked 589,792,833; native memory 135,161,459. Blocks 2,315,
promotions 319, regions 30, region exits 8,097,762, loop iterations 42,737,643,
code reads 13,194 and version checks 2,317 are unchanged. Invalidations,
compile failures, memory exits and profile errors are zero on this workload.
The comparison helper excludes only generated code size and genuine host
context-query counts; every other statistic is compared exactly.
Native bytes decrease from 144,496 to 142,698. Context checks decrease from
12,733,899 to 3,879,324 because scheduling no longer re-queries an executable
context that cannot change between its native calls; no fake checks were added.

Implementation:

- `RunQuanta` schedules at most sixteen **separate native calls**, each with its
  own <=64-instruction budget, private-stack switch, Go return and progress
  validation. It borrows one arena lock/view and aggregates host bookkeeping.
  `Run` remains one quantum and `Step` remains one block. Cold discovery,
  installation, promotion and slow memory/syscalls occur outside serialization.
- Ready publication generations prevent scheduling past cold region work. Only
  native non-executable stores can occur between scheduled calls in this
  single-threaded memory model, so the initial context proof remains valid.
  A fault, miss, zero progress, ceiling or cold target stops scheduling.
- Native loop predicates fuse compares/flag production with their immediate
  branch; boolean inversion is explicit. Exit-only SSA DAGs are reconstructed
  natively, with reverse leaf-lifetime propagation and per-exit memoization.
  Loop phis and ordinary uses remain hot; addition HI/LS preserves carry/zero.
- Separate call-local read/write page facts reuse full-page canonical/backing/
  permission proofs, never loaded memory values. Width guards remain per access
  and stores still check generation exhaustion. Facts reset per native call.
  Scratch grows by 32 bytes inside the same 17,408-byte frame/64 KiB stack.
- Compact amd64 immediate/checked-memory compares and mask tests retain all
  dispatcher guards. Successful transitions consume their own budget SUB flags
  immediately; no unrelated ambient flags are used.
- Architectural same-page pair reads/writes use one fully checked backing page.
  One version is allocated before either store. Cross-page accesses retain full
  prevalidation and version-exhaustion faults do not partially commit a pair.
  Pairs remain in the architectural handler, not native guest-register callbacks.

Exploratory single runs progressed through 1.19 (predicate fusion), 1.15
(two-level scheduler), 1.12 (ready publications), 1.13 (boolean inversion),
1.12 (proof reuse), 1.07 (aggregate scheduler), 1.04 (separate page facts),
1.04 (invariant publication/register pool), 1.03 (compact compares/masks),
1.02 (compare-memory), 1.01 (immediate predicates), 1.01 (budget flags),
and 0.96 (same-page pairs). They are exploratory, not isolated effect sizes.
Private native target caches with 64 and 256 records measured 1.20 and 1.17
and were **discarded completely**. No executable-pointer cache or extra context
field remains. All raw trials, paired runs and both profiles are retained in
`aarch64-coremark-results.json`.

Final profile: 102 samples (99 main PMU group, 3 separate group), zero lost
samples. The main group places about 28.59% in the native dispatcher, 8.68%
in prepared batch calls and 7.47% in runtime quantum selection. These short
samples are diagnostic, not precise benefit attribution. Raw reports are
`aarch64-coremark-profile-quanta-trial.txt` and
`aarch64-coremark-profile-subsecond.txt`.

New correctness coverage checks actual numbers of native calls for 257/10,000
instruction ceilings, the sixteen-call cap, ready-generation stops, closed
owners, cleanup/Close serialization and a precise store fault in the second
quantum. Independent arithmetic/logical flag oracles cover all sixteen
conditions, 32/64-bit widths, register/immediate operands, inversion and spilled
loop phis/cold maps. Page-proof tests cover write-only versus read permissions,
full noncanonical tags and cross-page widths on cache hits. Archive tests compare
`RunQuanta` against architectural stepping with odd/large exact ceilings and
revalidate mutations between schedules. Pair tests cover little-endian bytes,
unaligned/boundary offsets, aliases, read/write revocation, one generation,
cross-page partial faults and version exhaustion.

Runtime/runimage tests, embedded emulator package tests and both compiler builds
pass. The final required `repo.preflight()` passed in 29 seconds against its
unchanged 60-second budget, with no timeout, stderr or output truncation. Native
execution evidence is Linux/amd64 only; arm64 emitter selection is not hardware
execution evidence.
The 16-instruction trace limit, 64-instruction native-call quantum, 4,096-block
cache, 8 MiB code, 64 MiB guest RAM and 64 KiB private stack remain unchanged.

## Valid-duration scoring follow-up (October 7, 2026)

Three same-ELF pairs at 100,000 iterations per seed set now establish a native
median CoreMark 1.0 score of **2,116.10 iterations/second**, versus QEMU's
**7,591.34**. Native performance intervals are 47.24–47.29 seconds; QEMU
performance intervals are 13.16–13.18 seconds. Both required seed sets exceed
ten seconds in every scored run, validate successfully, and match all ten CRC
values between engines. QEMU remains about **3.59 times faster**.

Precise scores use performance-seed microsecond ticks, not integer-truncated
guest rates, validation-seed rates or total process wall time. These are local
emulator scores, not certified/submitted or AArch64 hardware results; no CPU
pinning or full host characterization was performed. See
`aarch64-coremark-score.md` for flags, provenance, all intervals, limitations and
reproduction, and `aarch64-coremark-score-results.json` for complete raw logs.
Earlier diagnostic results above are unchanged and remain short-run diagnostics.
