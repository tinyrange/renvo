# Freestanding Linux AArch64 CoreMark port

Original integration code for the unmodified EEMBC CoreMark sources at revision
`d5fad6bd094899101a4e5fd53af7298160ced6ab`. This directory does not vendor the
upstream benchmark. Retain its LICENSE.md (Apache-2.0 and trademark acceptable
use terms), README.md and source copyright notices alongside local copies.

Copy the five `core_*.c` files and `coremark.h` from that revision unchanged to
`sandbox/aarch64-coremark/`, and these three port files into its `port/` directory.
The existing approved `coremark.build(iterations)` workflow supplies identical
flags to every translation unit. No compiler or check workflow needs editing.
The tools are `aarch64-linux-gnu-gcc`, `aarch64-linux-gnu-objdump`, and
`qemu-aarch64`. Sources are user-provisioned, not downloaded by this port.

The entry point calls upstream main twice: once with performance seeds
(0,0,0x66), then validation seeds (0x3415,0x3415,0x66). Each interval uses the
requested iteration count and the standard 2000-byte total dataset. Seeds are
volatile globals; algorithms, CRC checks, and run-duration checks are unchanged.
The port uses only Linux write, clock_gettime(CLOCK_MONOTONIC), and exit.
Its bounded printf implementation supports the upstream integer-mode output;
unsupported formats or syscall failures terminate with status 125.

`Total ticks` are microseconds. `HAS_FLOAT=0` selects upstream integer seconds
and integer rates, which truncate: derive precise diagnostic elapsed time from
ticks, not the printed integer rate. Do not mistake a short CRC run for a valid
score. Each required seed set must run for at least ten seconds, and all CRCs
must match. Upstream main returns zero even when printing validation errors;
exit status alone is not a correctness or scoring check.

Expected seed/list/matrix/state CRCs:

- Performance: e9f5 / e714 / 1fd7 / 8e3a.
- Validation: 18f2 / e3c1 / 0747 / 8d84.

The final CRC depends on the iteration count and is compared across engines.
Run the identical ELF under all engines, without rebuilding between them.
Compilation, loader startup, and output are outside each guest timed interval;
`/usr/bin/time` includes them and both seed-set runs. Local measurements are recorded in the PR; implementation limits are in
`docs/aarch64-user.md`.
