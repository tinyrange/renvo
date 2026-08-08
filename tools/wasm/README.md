# WebAssembly compiler host

`cmd/renvowasi` is the size-constrained, self-hosted Renvo frontend for WASI.
It discovers and compiles a package to the canonical Renvo unit format. A
separate fixed-target backend module consumes that unit and emits a runnable
WASI command-line application. Browsers and command-line hosts compile and
cache the two WebAssembly modules independently, so adding another backend
does not increase the frontend module.

The production build uses the standard Go bootstrap and Renvo itself. TinyGo,
LLVM, Binaryen, and other post-link optimizers are not part of the build.

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

The compact frontend interface accepts `-o`, repeatable `-tags`, and one package path.
Its target is fixed to `wasi/wasm32`; `-s` and `-emit-unit` are accepted as
no-ops for compatibility. Human-readable, source-mapped diagnostics remain a
host/UI responsibility so that presentation code is not linked into every
compiler worker.

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

## Browser playground

Put both modules beside `tools/wasm/browser/index.html`, or pass their URLs in
the query string, then serve the repository over HTTP:

```sh
python3 -m http.server 8000
# http://localhost:8000/tools/wasm/browser/?compiler=/sandbox/wasm/renvowasi.wasm&backend=/sandbox/wasm/renvowasi-backend.wasm
```

The page keeps both compiled `WebAssembly.Module` objects in a worker and
creates fresh WASI instances for each command. The modules share a virtual
workspace only through the canonical unit. The worker returns per-phase timing,
peak linear-memory size, diagnostics, and downloadable application files. This
boundary is intended to grow into the IDE worker protocol and can load another
target backend without replacing the frontend.

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
