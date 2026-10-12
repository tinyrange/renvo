# AArch64 CoreMark >8,000 achieved — October 12, 2026

## Measured result

The final implementation exceeded **8,000 iterations/second on the performance
seed in both interleaved repetitions**. Its aggregate is **8032.78 iterations/s**.
The validation seed straddles 8,000 and aggregates to **7971.24**; this is not a claim
that both seeds consistently exceed the target.

These are local RFE emulator results on the existing Linux/amd64 host executing
an AArch64 guest, not physical AArch64 CPU scores or EEMBC-certified/submitted
results. The unchanged 100,000-iteration guest runs both seed sets.

| A/B/B/A run | Performance iter/s | Validation iter/s |
| --- | ---: | ---: |
| Baseline A1 | 7686.50 | 7656.84 |
| Owned pin graph B1 | **8057.98** | 8004.04 |
| Owned pin graph B2 | **8007.74** | 7938.70 |
| Baseline A2 | 7730.25 | 7677.31 |
| Aggregate baseline | 7708.31 | 7667.06 |
| Aggregate retained | **8032.78** | **7971.24** |
| Relative gain | **+4.21%** | **+3.97%** |

A is the previously retained register-result loop ABI with per-call pinning.
B adds the revalidated owned pin graph; compiler output and guest work are
otherwise identical. The aggregates divide total iterations by total raw
microsecond ticks across the two repetitions. The port's integer-second printed
rate is not used. All eight scored seed intervals exceeded ten seconds, all ten
CRCs matched in each trial, and all native execution counters were identical.

The margin above 8,000 is only **0.41%**. Two unpinned local repetitions per
variant do not establish a confidence interval or guarantee every future run
will exceed 8,000. The fresh baseline aggregate is not the earlier 7,825.49
checkpoint: the +4.21% improvement uses only this new interleaved comparison.
An earlier owned-pin prototype measured 8,180.37/8,100.71 before final cleanup;
it is saved for transparency but excluded from the final aggregate.

## Retained change

Profiling showed repeated runtime pin/unpin work at native session boundaries.
The RFE runtime now selects pin reuse only for host-owned contexts already
registered in its existing code-versioned ownership map. General contexts
retain the original per-call pinning. No benchmark-specific dispatch was added.

`internal/runimage/pure_session.go` retains at most **one pinned graph per arena**:
the current context, clock, each of the 64 data/epoch pointer pairs, the staging
state/image, private stack, entry table, prepared targets, and fault table.
Every call compares context and pointer identities under the arena lock.
Replacement unpins the old graph before pinning the new one; incomplete pin
initialization has cleanup. This amortizes GC pin bookkeeping, not execution
admission or guest-memory validation.

On every invocation the implementation still:

- Checks original state/context/page alias conditions; aliased layouts retain
  the conservative <=64-instruction assembly boundary.
- Copies current numeric page metadata and issues a fresh native admission epoch.
  Equal pinned pointers do not authorize changed permissions, page numbers,
  descriptors, budgets, or code versions.
- Enforces the existing 65,536-instruction foreign-session ceiling and validates
  exact retirement, remaining budget, memory count, and status on return.
- Clears all borrowed integer addresses in the public context and staging image.
  Retained GC pins are distinct from borrowed native execution views.

Pins are retained only after nonzero, fault-free, near-budget completion
(fewer than 256 instructions remaining). They are released on pointer/context
refresh, installation, ordinary/compatibility execution, invalid sessions,
faults, zero progress, substantial incomplete progress, and arena closure.
A short nonzero link miss within that near-budget window can retain pins;
the next call still performs fresh semantic checks. Owners must close the arena
when done, as required for its existing native allocations.

No session limits, instruction ceilings, resource gates, CoreMark sources,
port, compiler flags, seeds, timing interval, or guest ELF were changed.
The previous compiler register-result ABI remains in place; there is no further
retained compiler change in this pin-cache comparison.

## Verification

- All scored trials and the fresh diagnostic profile matched both fixed CRC sets.
- Every run retired exactly **59,156,887,738** guest instructions, with
  **13,687,594,350** native memory operations, **827,560** generated native bytes,
  **208** regions, **2,027,796,791** region exits, **4,730,437,296** loop iterations,
  and **908,864** native sessions.
- The actual-native lifecycle regression covers unchanged-graph reuse across GC,
  data/epoch/clock replacement, inactive-slot replacement, context replacement,
  permission and page-tag faults, stale-descriptor rejection with a small
  budget, installation, invalid/ordinary sessions, state aliases, and closure.
- Canonical preflight passed in **28 seconds against the unchanged 60-second
  budget**, including generated-source checks, tracked package tests, and
  backend/frontend test compilation. This is preflight, not the full merge gate.
- The final runner was rebuilt from the retained sources and the fresh profile
  was collected with that runner. Processes had no timeout or output truncation.

The new tests are `internal/runimage/pure_owned_session_test.go` and its
external-test opaque pin-identity helper, `owned_pins_export_test.go`.

## Diagnostic profile and rejected experiments

The same-run final profile has **2,740 exclusive IP samples**, **742** in the
native dispatcher (**27.08%**), and zero reported lost samples. It sampled
`pinOwnedSession` 14 times and no `runtime.(*Pinner).Pin`,
`runtime.(*Pinner).Unpin`, or `runtime.setPinned` instructions. The previous
2,647-sample profile contained 32 exclusive samples in those three runtime
symbols. This is supporting diagnostic evidence, not an isolated causal
measurement; separate CPU-event-weighted percentages were not added together.
Profiler timings are excluded from the unprofiled scored comparison.

Read/write fact pairing, resident pointer bias, and promotion thresholds of 128
and 32 did not establish useful gains and were reverted. Relaxing dispatcher
callee-register preservation passed the guest CRC trial but failed a broader
native differential test, so it was also reverted. The ABI preserves its
original required registers; no failing optimization is retained.

## Artifacts and workspace state

The `aarch64-coremark-8000-achieved` files contain the raw JSON, complete preflight,
CPU report, exclusive IP samples, matching native metadata, and a HEAD-relative
integration diff. That diff includes the previously retained register-result
integration; it is not solely a diff of the new cache against A. Exact A/B copies
of the four pin-cache integration files are retained under
`sandbox/owned-pins-comparison-2026-10-12/` for diagnostic reproducibility.

The live binary snapshot is `sandbox/aarch64-coremark/native-code.bin`; future
profiles overwrite it. Use only metadata and samples from the same run when
choosing native instruction offsets.

The original working tree and prior user changes are preserved. Everything
remains local: no commit, push, or PR. No fresh QEMU superiority claim is made.
