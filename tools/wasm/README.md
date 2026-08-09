# WebAssembly compiler host

`cmd/renvowasi` is the size-constrained, self-hosted Renvo frontend for WASI.
It discovers and compiles a package to the canonical Renvo unit format. A
separate fixed-target backend module consumes that unit and emits a runnable
WASI command-line application. Browsers and command-line hosts compile and
cache the two WebAssembly modules independently, so adding another backend
does not increase the frontend module.

The production build uses the standard Go bootstrap and Renvo itself. TinyGo,
LLVM, Binaryen, and other post-link optimizers are not part of the build.
The transient self-hosted compiler is built for the current supported host,
including Linux x86, x86-64, ARM, and ARM64, macOS ARM64, and Windows x86,
x86-64, and ARM64.

Build from the repository root:

```sh
tools/wasm/build.sh \
  sandbox/wasm/renvowasi.wasm \
  sandbox/wasm/renvowasi-backend.wasm
```

Run it with the repository preopened as its workspace:

```sh
node tools/wasm/run.mjs --workspace . sandbox/wasm/renvowasi.wasm -- \
  -o sandbox/wasm/renvowasi.unit -tags renvo_wasi_frontend ./cmd/renvowasi
```

The compact frontend interface accepts `-o`, `-t`, repeatable `-tags`, and one
package path. Built-in targets are recorded directly in the canonical unit.
Custom targets also pass the resolved definition hash and descriptor version;
the matching lazy backend remains responsible for the actual code generation.
`-s` and `-emit-unit` are accepted as no-ops for compatibility. Human-readable,
source-mapped diagnostics remain a host/UI responsibility so that presentation
code is not linked into every compiler worker.

Run the complete frontend/backend pipeline from Node:

```sh
node tools/wasm/compile.mjs --workspace . \
  sandbox/wasm/renvowasi.wasm \
  sandbox/wasm/renvowasi-backend.wasm -- \
  -s -o sandbox/wasm/app.wasm ./tools/wasm/testdata
node tools/wasm/run.mjs --workspace . sandbox/wasm/app.wasm --
```

`-emit-unit` stops after the frontend. `-arena-size` is host policy forwarded
to the backend and becomes the generated application's arena; for example the
frontend itself uses `-arena-size 167772160`.

Profile small, medium, and self-hosting builds in fresh Node processes:

```sh
node tools/wasm/profile.mjs sandbox/wasm/renvowasi.wasm
node tools/wasm/profile.mjs sandbox/wasm/renvowasi.wasm --check
node tools/wasm/profile.mjs sandbox/wasm/renvowasi.wasm \
  --backend sandbox/wasm/renvowasi-backend.wasm --check
node tools/wasm/profile.mjs sandbox/wasm/renvowasi.wasm --json > sandbox/wasm/profile.json
```

`--check` enforces a module no larger than 2 MiB and a self-host frontend run
under one second. Peak RSS is the Node process high-water mark. Linear memory
is the WASM reservation and is normally higher than resident memory because
V8 commits pages on demand. The WASI profile reserves a 160 MiB compiler arena
and permits a 6 MiB generated unit.

## Browser editor

Build the deployable static bundle and serve it over HTTP:

```sh
tools/wasm/build-browser.sh sandbox/wasm/browser
python3 -m http.server 8000 --directory sandbox/wasm/browser
# http://localhost:8000/browser/
```

The GitHub Pages layout also places the editor at the bundle root:

```sh
tools/wasm/build-browser.sh sandbox/wasm/pages pages
python3 -m http.server 8000 --directory sandbox/wasm/pages
# http://localhost:8000/
```

`.github/workflows/pages.yml` builds that layout after every push to `main`
and deploys it through GitHub Pages. The checked-in `CNAME` publishes the site
as `renvo.dev`; repository Pages settings and the domain's DNS records must
also name that domain before the first deployment.

The page loads the compact frontend and language-service modules at startup.
Backends are fetched and compiled only when their target is first built. One
shared native module covers every advertised desktop and VM target; WASI has
its size-optimized backend; ESP32-C6 and ESP32-S3 each use a prepared custom
backend generated from their checked-in RTG definitions. A target change does
not replace or reload the frontend.

Library metadata is small and loaded once. Standard-library, Forms, and ESP32
platform source is fetched only when it is browsed or imported, including its
dependencies. The same files are then passed to continuous diagnostics,
completion, signature help, definition/reference navigation, and compilation,
keeping editor and compiler views of the workspace consistent. Selecting a
catalogued `package main` example makes it the build root and selects its board
target.

Each command runs in a fresh WASI instance inside the worker. Build results
include per-phase timing, peak linear-memory size, diagnostics, and downloadable
artifacts. WASI command artifacts can run directly in the terminal panel with
arguments and preloaded standard input. For ESP32-C6 and ESP32-S3, Flash & Run
converts Renvo's ELF to the documented Espressif app-image format, writes the
app partition through WebSerial or WebUSB, reboots the board, and attaches the
terminal as a serial monitor. The transport picker uses WebUSB on Android and
WebSerial on desktop, where the operating-system CDC driver owns the control
interface required for reset. It remembers an explicit valid choice and falls
back to the available browser API. Both transports require an HTTPS or localhost
origin and a Chromium-based browser. The terminal reports build, flash, and
combined elapsed time. Native artifacts remain downloads.

The ESP ROM-loader implementation is first-party and has no third-party
runtime dependency. Its protocol behavior was checked against Espressif's
Apache-2.0-licensed [esptool-js at commit c2956c5](https://github.com/espressif/esptool-js/tree/c2956c5aac35d7ef24d614b734590263662eb8d8),
and against Espressif's public serial-protocol and firmware-image
documentation. No esptool-js source is copied or bundled.

The browser shell uses Monaco Editor 0.56.0 and keeps the small editable
workspace in browser local storage. Press Ctrl/Cmd+Enter to build or F5 to run
a WASI or ESP target; Run automatically builds first when the artifact is missing or the
workspace, target, or compiler arguments changed. Press Ctrl/Cmd+S to save the
current workspace and Ctrl/Cmd+J to toggle the panel. Monaco is loaded from
jsDelivr for now. Pass
`monaco=/assets/monaco/min/` to use a self-hosted copy in production.

On phone-sized screens the desktop chrome becomes full-screen Files, Code, and
Console workspace views. Selecting a source opens the editor directly; a
persistent Target action opens full-screen target selection when needed. Flash
opens a full-screen progress and serial-console view. WebUSB/WebSerial becomes
a touch choice in the target view rather than a compact dropdown. Read-only
library sources can replace the locally persisted playground `main.go` through
the editor's Copy into main.go action without changing the bundled original.

## Reference result

On Node v20.19.1, Linux/x64, and an Intel i7-13620H, the self-hosted frontend is
1,961,115 bytes and the separately cached backend is 837,097 bytes. The
frontend compiles its own package in about 500 ms; the backend turns that unit
into a runnable compiler in about 330 ms, for an approximately 832 ms complete
self-host pipeline. The frontend reserves about 171 MiB of linear memory and
peaks around 130 MiB resident. These figures are comparison data; the enforced
portable gates are a frontend no larger than 2 MiB and self-hosting under one
second.

The main backend size win comes from lowering virtual stack-frame accesses to
a per-routine frame-base local plus WASM memory offsets. That avoids rebuilding
`FP-offset` for every load and store while preserving the backend's ordinary
frame convention. Calls and returns are direct WebAssembly operations,
reducible control flow is emitted as structured blocks and loops, and only
irreducible graphs retain the dispatcher. The old virtual instruction stream
is therefore an internal lowering format, not an interpreter embedded in the
generated application.
