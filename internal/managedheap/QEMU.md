# Linux QEMU managed-heap validation

## Result

- **Managed opt-in: 33/33 PASS**, no missing or skipped targets.
- **Unchanged arena smoke: 33/33 PASS** after the adapter changes.
- All 33 installed emulators executed the workload. Managed tests make isolated
  copies and change only the selected target's memory strategy to `desktop`.
- Ordinary target definitions still default to arena. Experimental RBEs remain
  standalone; no target-name dispatch was added to the common collector.

## Per-target results

| QEMU suffix | Arena smoke | Managed opt-in |
| --- | --- | --- |
| `aarch64` | PASS | PASS |
| `aarch64_be` | PASS | PASS |
| `alpha` | PASS | PASS |
| `arm` | PASS | PASS |
| `armeb` | PASS | PASS |
| `hexagon` | PASS | PASS |
| `hppa` | PASS | PASS |
| `i386` | PASS | PASS |
| `loongarch64` | PASS | PASS |
| `m68k` | PASS | PASS |
| `microblaze` | PASS | PASS |
| `microblazeel` | PASS | PASS |
| `mips` | PASS | PASS |
| `mips64` | PASS | PASS |
| `mips64el` | PASS | PASS |
| `mipsel` | PASS | PASS |
| `mipsn32` | PASS | PASS |
| `mipsn32el` | PASS | PASS |
| `or1k` | PASS | PASS |
| `ppc` | PASS | PASS |
| `ppc64` | PASS | PASS |
| `ppc64le` | PASS | PASS |
| `riscv32` | PASS | PASS |
| `riscv64` | PASS | PASS |
| `s390x` | PASS | PASS |
| `sh4` | PASS | PASS |
| `sh4eb` | PASS | PASS |
| `sparc` | PASS | PASS |
| `sparc32plus` | PASS | PASS |
| `sparc64` | PASS | PASS |
| `x86_64` | PASS | PASS |
| `xtensa` | PASS | PASS |
| `xtensaeb` | PASS | PASS |

## Checks executed on every target

- Repeated allocations totaling more than the 64 KiB cap, requiring reclamation.
- Live globals, interior pointers and a 24-node graph retained across pressure
  and explicit Collect(). The graph puts a pointer at a four-byte offset on
  32-bit machines, exercising native-width scanning rather than eight-byte-only
  scanning.
- A 1 MiB request under a 64 KiB cap panics and is recoverable; subsequent
  allocation/collection and root validation continue working.
- One-byte environment and compiled limits fail at startup with status 2.
- Malformed and overflowing environment values fail with exact diagnostics.
  On 32-bit targets, the out-of-width `1TiB` suffix is also rejected.
- Raising a one-byte compiled limit to 64 KiB via the environment permits the
  entire pressure program.

The test enforces unchanged 60-second per-build/run deadlines and exact
output/exit codes. It reads the existing matrix, including the xtensaeb CPU
argument. Missing emulators, unsupported definitions and crashes are failures,
not skips. The final run against the regenerated compiler bundle passed all
33 rows in 76.74 seconds total; the earlier expanded-matrix run took 44.41 s.
These totals include preparation/build work and are not guest throughput scores.

Reproduce from the repository root:

```sh
./tools/qemu-smoke/run
RENVO_QEMU_MANAGED=1 go test ./frontend_tests -run '^TestManagedHeapQEMU$' -count=1 -v -timeout=35m
```

The managed test writes `sandbox/agent-profiles/managed-qemu.json`. Captured logs
for this work are `managed-qemu.log`, `arena-qemu.log`, and
`preflight-managed-ports.log` in that directory. QEMU remains opt-in, outside the
one-minute preflight workload.

## Implementation notes

The shared collector now scans and zeros native 32- or 64-bit words while
retaining aligned headers. Runtime adapters own register snapshots, initial
stack/environment layout, mapping granularity, syscall numbers and error
normalization. MIPS O32/N32/N64, SPARC stack headers, s390 old_mmap argument
blocks, and HPPA's existing private stack are handled in their definitions.
The shared collector copies in standalone RBEs are checked against the canonical
source by `TestManagedEmbeddedCollectorConsistency`.

## Preflight investigation

Production RTG lookup/generation optimizations avoid decoding every sequence
when looking up a single hook and avoid rebuilding architecture symbol tables
inside emission loops. No tests, timeouts, memory gates, parallelism settings or
budgets were weakened.

Earlier profiled package wall times (single runs, not controlled benchmarks):

| Package | Before lookup changes | After lookup changes |
| --- | ---: | ---: |
| internal/rtg | 15.283 s | 12.958 s |
| internal/backendjit | 19.061 s | 20.461 s |

After the platform ports, isolated profiles measured RTG at 10.122 s and
backendjit at 22.921 s. Backendjit has not shown an improvement. Its remaining
work includes parsing/checking/preparing compiler source, not only RTG lookup.

The first preflight after regeneration passed functional checks but took **72 s**
and correctly failed its **60 s** budget. After package profiling and subsequent
builds, the full unchanged preflight passed in **48 s**. Rebuild/cache overhead
remains a timing risk: the passing result is not a cold-build guarantee.

Native Linux/amd64 managed stress, renamed definition-port, mapping-granularity,
and three-stage managed compiler self-host tests also pass. Managed self-hosting
still uses its existing 256 MiB budget and per-process deadlines.

## Qualification limits

These are executable correctness regressions, not exhaustive GC proofs, native
hardware qualification, throughput/pause guarantees, or compiler self-hosting
proof on all 33 architectures. The collector remains conservative and synchronous
on one native stack. Other OSes, switched stacks, concurrent mutators and native
FFI/object output remain outside this qualification. See `README.md` and
`STRESS.md` in this directory for the runtime contract and remaining limitations.
