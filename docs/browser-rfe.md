# Browser-hosted compiled Linux RFEs

This change adds a Renvo-produced browser/Wasm host for the existing AArch64
and x86-64 Linux machines. Hot guest blocks and regions execute in generated
WebAssembly sharing the emulator's linear memory, with architectural fallback
for unsupported operations and precise exits. It does not replace the native
host or advertise an interpreter as the compiled tier.

## Boundaries

- Immutable per-entry private proofs guard table dispatch, ownership, full PCs,
  state dimensions, instruction budgets and precise retirement.
- Memory operations retain permission, page identity, executable/watch and
  generation checks. Callback/refill paths publish architectural state and
  reload invalidated observations before retrying.
- Browser-only translation lifetimes and bounded caches are distinct from
  native platform implementations. Recycling closes authority before storage
  can be reused; coherent snapshots do not add a new unchecked cache.
- Browser gzip/ELF loading is bounded. Browser host syscalls are separate from
  ordinary and dynamically selected WASI syscalls.
- Compiler fixes cover ARM frame immediates, wide word/index semantics,
  complete-word comparisons, closure/bound-receiver lifetimes, blank globals,
  promoted scalar lifetimes and checked panic propagation. Parsed source
  semantics and regression programs accompany these fixes.
- Current main's managed-heap support, native dispatch/read-span work, ARM64
  elapsed-context restoration and portable host-session assertions are retained.

The ordinary boot/correctness gates keep their existing 300-second deadline,
two-billion scheduling bound, 256 MiB guest RAM and bounded host arenas/sessions.
No performance-policy reference or CI workflow is changed. The separate offline
CoreMark diagnostic harness uses preserved fixed inputs, two invocations,
complete PERF/VALID CRCs, host-monotonic serial timing and its original
performance-only bounds; those are not substituted for correctness checks.

## Build and run

See `tools/rfe-browser/README.md` for the builder, offline checker and optional
local/tailnet demo. Kernel and initramfs remain separate matching-architecture
assets, not committed binaries or embedded downloads. The fixture preparation
workflow and native Linux checks remain documented in `tools/rfe-linux/README.md`.

The optional benchmark/compiler-candidate scripts require their recorded cached
inputs, original binaries and provenance receipts. Their pinned historical
fingerprints are intentional experiment boundaries, not a claim that an
arbitrary fresh checkout can replay those scores. There are no checked-in
benchmark artifacts, generated browser pages, kernels or machine-local
configuration changes in this PR.

## Validation and performance claims

The PR is integrated on current main, rather than overwriting its intervening
compiler and RFE fixes with an old working-tree snapshot. Local preflight and
focused regression results are recorded in the PR description; full required
CI remains authoritative for merge readiness.

Historical retained ARM64 measurements were roughly 1,050–1,166 warm PERF host
iterations/s. The **2,000 host iterations/s objective has not been reached**.
Two later isolated compiler trials (adjacent wide-word store fusion and a
split-word local representation) did not establish a consistent speedup and
are **not included**. These historical measurements describe their exact
recorded source/compiler/input snapshots, not a new score for the integrated PR.

At 20,000 iterations, the unchanged ten-second minimum duration cannot certify
a score above 2,000 iterations/s; longer-iteration certification is separate.
No official EEMBC submission, cross-host performance guarantee or sub-ten-second
boot claim is made.
