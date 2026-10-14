# System-clock and page-table-watch repair

## Final measured results

Configuration: unchanged cached kernel, initramfs and CoreMark ELF; 100,000
iterations per PERF/VALID seed; logical CPU 0; 256 MiB; balanced32 cache;
new explicit `-clock realtime` mode. Benchmark code, compiler flags, advertised
clock frequencies and timed-port markers were not changed.

| Architecture | Old PERF median | Final PERF | Final VALID | PERF speedup |
|---|---:|---:|---:|---:|
| AArch64 | 1191.946 | 5080.900 | 5044.326 | 4.263x |
| x86-64 | 1065.081 | 4187.280 | 4188.966 | 3.931x |

Rates are iterations per **host** second. Both exceed 4,000. Final observations
are single runs versus historical three-run medians, not new repeated-run
medians. Both host and guest intervals exceed 10 seconds. Both seed CRC sets
and final CRCs match, both processes exit 0, no output is truncated, and guest
input hashes remain unchanged.

ARM PERF: 19.682 host / 19.279 guest seconds.
x86 publication PERF: 23.882 host / 23.261 guest seconds. The earlier clock-only
observation was 4,232 iterations/s; the final publication run also includes the
INVLPG and boot-bounds review corrections below.

## Root causes and fixes

### Retirement was being used as timer time

Clocks advanced from retired instruction counts. Native execution therefore
advanced guest time and interrupts independently of elapsed host time. Added
`internal/rfe/systemclock`: a shared host-monotonic oscillator for realtime
mode. PIT/TSC and ARM counters/deadlines use that oscillator. Idle waits do not
invent retirement. Cold decoding/compilation and arena reclamation pause VM
time while the vCPU is stopped; guest execution and device/idle work do not.
Deterministic prefix-based clocks remain supported.

### A region side exit is not a machine scheduling boundary

A region exit can continue in the native dispatcher. A clock snapshot held
through a whole quantum can waste 65,536 instructions in a short delay loop.
Added an explicit instruction scheduling-boundary property. Realtime counter
reads are isolated native leaves, excluded from linked/cyclic traces, and
return to the machine before following instructions. Regressions verify actual
native retirement/PC, frozen clocks, clock progress and trace formation.

### Blocked PIC requests forced single stepping

x86 reduced its native budget to one instruction whenever IRR contained an
unmasked request, even if IF was clear or an IRQ already in service. A reduced
test failed with exactly one retired instruction. Only a deliverable request
now needs the one-instruction boundary; blocked requests are preserved.

### Page-table watches had the wrong lifetime

Every physical page ever seen as a page table retained a write barrier forever.
When freed/reused as ordinary kernel data, each store invalidated native code.
This explained the ARM realtime regression, not an inherent ARM limitation:
17.44 million watched page-table writes and 40.40 million user link misses.

Full architectural TLB revocations now revoke native descriptors/code-context
stamps BEFORE retiring old table watches. New walks rewatch active tables;
executable-page watches remain protected. ARM capacity eviction occurs before
the new walk so the newly cached translation retains its table protection.

x86 preserves bounded root-tagged proofs across CR3 switches, so full-flush
retirement alone was insufficient. Watches now have cached-root/domain owners.
They are released only after the root proof is invalidated/evicted, and only
when no other live root owns the frame. Shared live roots and executable aliases
remain protected. Metadata is bounded by RAM pages and existing root slots;
root churn cannot append an unbounded global list.

After ARM's lifetime fix, page-table writes fell to 18,730 and user link misses
to 34,219 in the first successful 5,177 iterations/s run. Final shared-source
confirmation is 5,081 iterations/s. On x86, bounded-root retirement reduced
page-table writes from 14,032,942 to 762,248 and user link misses from 2,527,798
to 52,175; throughput rose from 1,566 to 4,232 iterations/s.

### Hardware A/D bookkeeping was treated as a mapping mutation

A separate x86 reproducer showed hardware accessed/dirty updates unnecessarily
invalidating executable mappings. A walker-only update changes only A/D bits,
revokes stale data descriptors, preserves dirty-bit write checks, and still
invalidates executable table aliases. Guest-authored PTE stores retain full
conservative invalidation behavior.

## Publication review corrections

A failing x86 reproducer showed that native descriptors refilled from an old
architectural TLB after a PTE store could survive INVLPG. Single-page invalidation
now revokes native descriptors and code-context stamps, but preserves table
watches belonging to other live architectural TLB entries. Unknown memory buses
fall back to a complete architectural/native flush. The final x86 score above
was confirmed after this correction, with all original CRC/hash/time checks.

Two additional failing reproducers exposed truncated bzImage-header slicing
and initramfs-placement underflow. Both inputs now return errors instead of
panicking. Portable Renvo-built hosts also retain the new owned-direct-loop API
as an unavailable-native fallback; stage0 and stage3 frontend checks pass.

## Merge-group ARM64 correction

The full merge-group suites exposed an ARM64-only native ABI bug that the
Linux/amd64 preflight could not execute: contextual elapsed capture used X1 as
scratch, although X1 also carried the loop's live memory-context pointer. The
next status write used the captured retirement count as an address. Restore
that pointer from the already-rooted shared frame before status initialization.
`TestARM64ElapsedLoopEntryKeepsMemoryContext` independently interprets the
emitted ARM64 entry prefix on every host; it failed before the fix with a store
to address `0xf`, then passed. Existing elapsed tests exercise full native
execution in the platform suites.

Two new tests also incorrectly expected large sessions on hosts without the
foreign-call bridge. They now retain exact progress checks at the original
64-instruction compatibility boundary. The folded-page test completes the same
100 iterations across those boundaries and still rejects refill/link misses.
Both tests exercise an actual state/clock alias locally to select compatibility,
in addition to the normal host-session path. The machine archives' console and
caller-slice tests likewise require exactly 64 instructions on compatibility
hosts, while retaining their larger-session assertions on bridge hosts and
unchanged timer/console bounds. No runtime limit or gate changed.

Correction validation: affected suites and three prerequisite backend corpus
programs pass; isolated preflight passes in 28 seconds / 60 seconds. Platform
correctness and performance results remain subject to the final merge group.

## Scope and compatibility

The validated performance configuration uses:

    -clock realtime -jit-cache balanced32

The existing CLI default remains `deterministic`. Original per-domain
correctness budgets, including the 2 MiB kernel arena, remain unchanged.
Small-cache realtime-default boot trials exceeded the original deadline; no
limits, harness settings or allocations were relaxed to hide that result.
The correction is validated in the hosted performance configuration, not a
claim that all clock/cache combinations now pass the original boot budget.

## Verification

- Complete AArch64 and x86 CPU/system archive suites: PASS.
- systemclock, engine and systemmemory suites: PASS.
- Original repair and final isolated publication preflight: PASS in 21 seconds
  under the unchanged 60-second budget (first isolated cold run: 35 seconds).
- Original two-architecture Linux gate: PASS for boot, exec, file I/O,
  pipelines and shutdown, with original default-cache/300s/2B limits.
- Safety regressions cover INVLPG, malformed boot inputs, table reuse, shared/cross-domain root ownership,
  identity reuse, executable aliases, active PTE writes, remaps/permissions,
  rewatching after capacity eviction, dirty-bit updates and bounded metadata.

The clock/page-watch repair itself did not change the backend emitter. This
publication also includes its prerequisite contextual elapsed-time IR, grouped
memory retirement and invariant-phi corrections with minimal backend regressions.
No resource gates were weakened. Publication was prepared in a clean main-based
worktree; unrelated original-checkout changes and local configuration are excluded.

## Evidence

Original clock-repair summary: docs/system-clock-repair-results.json.
Final x86 publication confirmation: docs/system-clock-repair-publication-x86-result.json.
Publication safety evidence: docs/system-clock-repair-publication-reproducers.txt.
Unchanged Linux/ARM compiler self-host gate: docs/system-clock-repair-arm-compiler-gate.txt.
Full provenance and corresponding serial/stderr logs are retained in the
original measurement workspace under docs/system-clock-repair-{aarch64,x86_64}
and the raw directories identified in the summary. They are not bundled into
this source PR.
Checks: docs/system-clock-repair-preflight.txt and -linux-gate.txt.
Failed-before/passed-after evidence: docs/system-clock-repair-reproducers.txt.

Source areas: machine archives, amd64 timestamp/walker support,
internal/rfe/systemclock, internal/rfe/systemmemory and engine scheduling/
translation boundaries. Approved measurement/kernel-inspection implementations
live outside AGENTS.star; configuration contains only thin named wrappers.
