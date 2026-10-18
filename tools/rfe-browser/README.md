# Compiled Linux RFEs in a browser

Both `linux-arm64-machine` and `linux-amd64-machine` can be built by Renvo to
`browser/wasm32`. The resulting HTML embeds the emulator WASM and its worker
host. Hot guest blocks and regions are compiled to additional WebAssembly
modules sharing the emulator's linear memory; the IR evaluator is not used as
the compiled tier. Architectural fallback remains available for unsupported
instructions and precise memory/privileged exits.

## Use the generated pages

1. Open `sandbox/rfe-browser/aarch64.html` or `x86_64.html` in a modern browser
   with WebAssembly, workers, BigInt, and WebAssembly reference tables.
2. Select **Kernel** and **Initramfs** from the matching architecture directory:
   `sandbox/rfe-linux/ARCH/kernel` and `initramfs.cpio.gz`.
3. Click **Boot**. The serial console reports the guest checks and shutdown.
   Select **Shell after checks** before boot to request the optional interactive
   console. Keyboard input and paste are forwarded to the serial console; Stop
   terminates the worker.

The OS assets are deliberately separate, not embedded in the HTML. Never mix
architectures. See `tools/rfe-linux/README.md` for pinned fixture provenance and
preparation. No network access is needed when selecting local assets manually.
The optional shell UI exists, but the automated gate validates the check-and-
shutdown path, not an interactive shell session.

## Build and validate

From a configured StarAgent workspace, use the approved workflows:

```python
go.build_compiler()
rfe.browser_build("aarch64")
rfe.browser_build("x86_64")
rfe.browser_check("aarch64")
rfe.browser_check("x86_64")
repo.preflight()
```

The CLI build entry point is `renvoemu build -host renvo -target browser/wasm32
-compiler PATH -o OUTPUT.html emulators/linux-arm64-machine.rfe` (substitute
`linux-amd64-machine.rfe` for x86-64). It emits the HTML and an adjacent JSON
build stamp with source/compiler/artifact hashes. Build the bundled compiler
first; do not use stale generated backend bundles. Normal native builds keep
the existing Go-host workflow.

The offline Node/Chromium checker in this directory requires installed
Node with WebSocket and Chromium. It installs nothing, verifies the pinned
cached Linux inputs and build freshness, and serves only loopback assets.
It records browser version, boot log, result JSON and screenshot beneath
`sandbox/rfe-browser/ARCH-latest.*`.

The original limits remain: 300-second boot deadline, two billion scheduling
attempts, 256 MiB guest RAM, 512 MiB emulator arena, bounded compiled arenas
and bounded sessions. The checker requires ordered INIT/EXEC/FILE/PIPE/PASS
markers **and** successful machine shutdown; printing markers alone is not a
pass. Browser startup probes execute actual generated WASM arithmetic, memory,
precise checkpoint and link-proof rejection paths. The x86 build also compares
compiled string operations with architectural reference execution.

## Implementation boundaries

- Pure integer WASM lowering includes flag operations and total division,
  with explicit memory/permission/page-identity/generation guards.
- A private WASM-table dispatcher checks immutable per-target proofs and
  retirement/budget accounting before continuing to successors.
- Proof instances share compiled code, not mutable authority: each captures
  independent immutable PC/entry/instruction-count/state-dimension globals.
  Full encoded proof bytes are still charged at each admission.
- Bounded decoded-instruction ownership and synchronous translation scopes
  prevent per-retirement/per-translation arena leakage.
- x86 REP string lowering executes one architectural iteration at a time and
  preserves exact count, direction, comparison flags and fault checkpoints.
- Portable bounded gzip/ELF loading supplies the browser x86 boot path; native
  loaders and native resource policies remain separate.

See `docs/browser-rfe.md` for implementation scope, validation and historical
measurement limitations. Current main-based validation is separate from the
original experimental snapshots.

## Tailnet demo server

The architecture selector is served at `http://100.66.0.124:8081/` by the
approved `rfe.web_start()` workflow. It preloads the matching kernel and Alpine
3.24.2 minirootfs-derived initramfs; no manual file selection is necessary.
The optional shell is selected before boot. Execution happens in the visiting
browser, not as a server-side VM.

The server binds only the specified tailnet address, not all interfaces. It
exposes the selector, two immutable browser artifacts, four checksum-verified
boot assets and `/healthz`. It rejects directory listing, traversal and writes;
it does not alter Tailscale configuration or firewall policy. `rfe.web_check()`
checks asset provenance and HTTP restrictions; `rfe.web_probe()` checks the live
fixed-address endpoints. The returned background process is session-owned and
can be stopped with its `kill()` method. It is not installed as a system service.
Port 8080 was already in use and its existing service was left untouched.

## Measure boot and shell latency

`profile.mjs` is a diagnostic separate from the shutdown correctness gate:

```sh
node --test tools/rfe-browser/profile.test.mjs
node tools/rfe-browser/profile.mjs aarch64
```

It checks the artifact stamp and cached fixture hashes, instruments an in-memory
copy of the page, and profiles the worker in an isolated Chromium process. It
timestamps the first kernel step, serial milestones, and an actual completed
shell command in the worker. Startup probes are excluded from kernel counters;
the existing guest init checks remain in the measured path. Asset transfer and
emulator initialization are outside `kernel_to_shell_roundtrip_ms`. Sampling
and timing hooks add overhead, so these are instrumented measurements.

Reports and raw CPU profiles go to `sandbox/rfe-browser-profile/`. To compare a
candidate without replacing the hosted artifacts, build it separately and pass
explicit artifact/report paths:

```sh
mkdir -p sandbox/rfe-browser-profile/candidate
go run ./cmd/renvoemu build -host renvo -target browser/wasm32 \
  -compiler sandbox/agent-bin/renvo-bootstrap -arena 536870912 \
  -o sandbox/rfe-browser-profile/candidate/aarch64.html \
  emulators/linux-arm64-machine.rfe
node tools/rfe-browser/profile.mjs aarch64 \
  --artifact sandbox/rfe-browser-profile/candidate/aarch64.html \
  --output sandbox/rfe-browser-profile/candidate/aarch64
```

Optional diagnostics:

- `--timing-only` omits CPU sampling and per-call timers, retaining exact serial
  milestones and kernel-step timing.
- `--translation-details` tracks bounded exact module identities, repeated
  guest PCs and duplicate installations, and names generated modules in CPU
  samples. Reports label tracking overflow and ambiguous PC associations.
- `--initrd-uncompressed` derives raw newc bytes in memory from the verified
  pinned gzip. Reports retain both hashes, sizes and host preparation time;
  the fixture remains unchanged.
- `--jit-cache NAME` explicitly selects an existing runtime cache preset:
  `default`, `balanced8`, `balanced32`, `kernel32`, or `balanced64`.
- `--engine-profile` enables bounded engine/mapping/exit snapshots on AArch64.
- `--compiler sandbox/PATH` verifies the artifact against an isolated compiler
  build instead of the default bootstrap; its path and hash appear in the report.
- `--progress` prints a compact progress update about every ten seconds.

The profiler retains the 300-second browser interval and guest instruction
limit. Cache overrides are recorded explicitly; the default and canonical
gates keep their existing memory budgets. Success requires a completed shell
response and, unless timing-only was selected, CPU capture. Failure reports
preserve available partial evidence. Profiling does not replace the
check-and-shutdown gate or compiler performance checks. SIGINT/SIGTERM cleans
up the profiler's own Chrome process group.
