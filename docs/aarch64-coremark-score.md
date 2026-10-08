# Valid-duration local CoreMark scores

Measured October 7–8, 2026, on the existing Linux/amd64 host, emulating a Linux
AArch64 guest. These are local emulator scores, not EEMBC-certified or submitted
results, physical AArch64 CPU measurements, or CoreMark/MHz results.

## Target confirmed on resumed working tree, October 8, 2026

**The local 5,000 target is reached in two fresh untraced trials.** On resumption,
the working tree already contained newer session-scoped admission caching and
four-way full-PC-tagged descriptor/proof lookup changes beyond the older report
below. This continuation rebuilt and validated those sources; it did not add a
new implementation or claim isolated speedups for individual changes.

The unchanged 100,000-iteration ELF ran sequentially native/QEMU/native:

| Engine/trial | Performance iterations/second | Validation iterations/second |
| --- | ---: | ---: |
| Native 1 | **5,348.42** | 5,341.04 |
| QEMU | 7,546.54 | 7,610.60 |
| Native 2 | **5,361.57** | 5,366.27 |

All ten CRCs match for each run, both seed intervals exceed ten seconds, and
processes complete without timeout or output truncation. Each native trial
retires exactly 59,156,887,738 guest instructions. Scores use microsecond ticks,
not the guest's integer-second `Iterations/Sec` line. These are two unpinned
local trials, not a median or EEMBC-certified results. **QEMU remains faster.**
Benchmark sources, port, flags, guest ELF and resource gates were unchanged.

The runner rebuild, session/prepared/linked tests, runtime native tests, and
unchanged preflight all pass.

A separate fresh profile also validates both CRC sets: 3,879 samples with zero
lost samples. The dominant `cpu_core` event attributes 25.14% to the native
dispatcher, 5.90% to the five-instruction region at guest PC `0x400dc8`, and 4.20%
to the six-instruction region at `0x400db0`. These are the next investigation
leads, not exact branch counts or guaranteed speedups. The profiled run is not
included in the scored trials above.

Raw process evidence: `aarch64-coremark-5000-confirmed.json`.
Fresh profile: `aarch64-coremark-profile-5000-confirmed.txt`.
Earlier experimental logs remain local and are not required to reproduce these results.


## Direct host comparison

On the same Intel Core i7-13620H, two direct x86-64 trials average **33,730.35**
performance iterations/second, compared with **5,355.00** for the emulator:
**15.88% of the direct-host rate**, or approximately **6.30x slower**. Both host
seed sets validate and run for more than ten seconds (1,000,000 iterations).
The final CRC depends on the iteration count and is not compared to the
100,000-iteration guest final CRC. Algorithm CRCs agree.

This is a scalar, generic x86-64 `-O2` comparison without vectorization, not
maximum tuned or whole-chip throughput. The algorithm sources are unchanged;
the target architecture, port and iteration count differ. Trials are unpinned.
Raw outputs and flags: `aarch64-coremark-host-comparison.json`.

## Reproduction and scope

See `../emulators/coremark-port/README.md` for the pinned upstream revision,
license/provenance requirements and AArch64 port. The corresponding direct-host
port is in `../emulators/coremark-host-port/`. Approved fixed workflows in
`AGENTS.star` build and run these local, user-provisioned benchmark sources.
Use the same guest ELF for native and QEMU; do not include profiled runs in
scored trials. Every seed must validate and exceed ten seconds before scoring.

The implementation retains bounded native traces, not a general optimizing CFG
engine. Cold blocks remain limited to 16 instructions, optimized regions to 256,
and supported Linux/amd64+cgo foreign sessions to 65,536. Other hosts/no-cgo
and alias compatibility use short calls. Direct chaining remains opt-in and
is disabled in these results. Resource gates and benchmark algorithms are unchanged.
