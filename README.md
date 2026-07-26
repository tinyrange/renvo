# Renvo

Renvo is a compact, ahead-of-time compiler for a practical Go subset. It is
built around a small, platform-independent core that can emit unusually small
native programs without depending on the Go toolchain, a system linker, or a C
runtime on the target machine.

The project is designed for retargeting. The same frontend can produce Linux,
Windows, macOS, and WASI executables today, while the backend boundary and
bring-up tooling are intended to make much smaller systems—down to constrained
microcontrollers—reasonable future targets.

Renvo is pre-1.0 software and is not a drop-in replacement for the Go compiler.
It deliberately ships a small runtime and standard library, accepts a broad but
incomplete Go language subset, and reports unsupported toolchain features
explicitly.

## What makes Renvo different

- **One-file toolchains.** Release frontends embed the backend and supported
  standard-library sources. A single executable can compile a normal Go module
  for any supported target away from the repository.
- **Tiny output.** Renvo writes executable formats directly and avoids a
  general-purpose runtime, linker, and object-file pipeline where a target does
  not need them.
- **Cross-target by construction.** Target selection is an ordinary compiler
  option, not a host toolchain installation problem.
- **A narrow backend contract.** The frontend lowers linked source into a
  versioned Renvo unit. Native backends, future C output, and board bring-up
  tools can share that stable handoff.
- **Bootstrap-focused testing.** The backend regression corpus and frontend
  acceptance corpus exercise self-hosting as well as generated programs. The
  omnibus tooling is designed to validate a new target with one compiled
  artifact when a conventional test suite is impractical.

## Supported targets

| Target | Output |
| --- | --- |
| `linux/amd64` | Static position-independent ELF executable (ASLR) |
| `linux/386` | Static PIE ELF executable |
| `linux/aarch64` | Static PIE ELF executable with ASLR |
| `linux/arm` | Static PIE ELF executable |
| `windows/amd64` | PE executable |
| `windows/386` | PE executable |
| `windows/arm64` | PE executable |
| `darwin/arm64` | Mach-O executable |
| `wasi/wasm32` | WebAssembly module |

The frontend supports packages and modules, local replacements, build tags and
target-specific files, `//go:embed`, and an offline module cache. Language
coverage includes ordinary control flow, methods, maps, interfaces, closures,
defer/panic/recover, arrays and slices, complex values, and the builtins needed
by Renvo itself. Generics, goroutines, channels, `select`, and cgo are currently
out of scope.

## Build and try it

A host-built frontend uses the standalone backend during development:

```sh
go build -o renvo-backend ./backend
go build -tags renvo_bundle -o renvo ./cmd/renvo

RENVO_BACKEND="$PWD/renvo-backend" ./renvo \
  -t linux/amd64 -o hello ./path/to/hello-package
```

Running `renvo` with no arguments or with `--help` prints the complete command
reference and target list.

For small scripts, the opt-in `run` command supplies `package main` and
`func main()` automatically:

```go
import "os"

print("hello ")
print(os.Args[1])
print("\n")
```

```sh
renvo run hello.go -- world
```

On Linux, `run` maps the linked image into the compiler process, applies
read/write/execute protections per segment, and calls its entry point on an
isolated native stack. It does not write an executable or launch a child
process. Windows and macOS use their native process APIs for the current
implementation.

An experimental REPL is implemented as a pure Renvo application on top of that
linked-image entry point:

```sh
./renvo-standalone -tags renvo_bundle -t linux/amd64 -s \
  -o renvorepl ./cmd/renvorepl
./renvorepl
```

The REPL accepts multiline expressions, statements, imports, and declarations.
Expressions are printed automatically. Successful imports, declarations, and
assignments become successive in-process linked-image generations.
Stable symbol cells preserve variables, pointers, and closures while earlier
statements are never executed again. `:history`, `:source`, `:reset`, and
`:quit` inspect or control the live linker session.

`-emit-image` exposes the same versioned `RNVI` linked-image transport without
executing it. The transport identifies the target and native format, validates
the payload, and presents code/data segments plus a relative entry point to
loaders.

To turn that bootstrap build into a fully standalone Renvo executable:

```sh
RENVO_BACKEND="$PWD/renvo-backend" ./renvo \
  -tags renvo_bundle -t linux/amd64 -s \
  -o renvo-standalone ./cmd/renvo
```

`renvo-standalone` contains the standard library and an in-process backend. It
can be copied to an empty directory and used without a Go installation,
repository checkout, adjacent data files, or backend process.

Useful development overrides are:

- `RENVO_BACKEND`: backend executable used by a host-built frontend.
- `RENVO_STDROOT`: standard-library source tree; defaults to the embedded copy
  in bundled builds.
- `RENVO_MODCACHE`: read-only, pre-populated module cache for offline
  dependencies.

Renvo never fetches dependencies while compiling. Use a local `replace`, a
module `vendor` directory, or populate `RENVO_MODCACHE` beforehand.

## Repository layout

```text
cmd/renvo/          command-line compiler
cmd/renvorepl/      experimental pure-Renvo interactive compiler
cmd/renvoide/       beta graphical development environment
internal/           parser, checker, loader, lowering, linker, and driver
std/                Renvo's target standard library
forms/ and ide/     reusable IDE and UI packages
frontend_tests/     package, diagnostic, self-host, and standalone acceptance tests
backend/            code generators, runtime shell, target descriptions, and backend tests
```

The root is the frontend module, published under the canonical path
`renvo.dev`. Backend implementation details are isolated under `backend/`.
The frontend/backend wire format is specified in
[`backend/unit/schema.json`](backend/unit/schema.json) and documented in
[`backend/docs/unit-v1.md`](backend/docs/unit-v1.md).

## Development and testing

The repository contains independent programs in `backend/tests/` and generated
corpus modules in `frontend_tests/`, so `go test ./...` is intentionally not the
whole-project command.

Useful focused checks are:

```sh
go test ./internal/... ./std/... ./cmd/...
go test ./backend/unit ./backend/target ./backend/bringup ./backend/omnibus/...
go test ./frontend_tests
go test -run '^(TestCompileTests|TestUnitFrontendCompileTests)$' ./backend
```

GUI frontend compile time and Go heap allocations have dedicated cold,
unchanged, and edited-project benchmarks:

```sh
go test ./cmd/renvoide -run '^$' -bench '^BenchmarkGUIFrontend' -benchmem
```

The GitHub Actions workflow runs the complete backend matrix, resource and
performance gates, self-hosted frontend corpus, bundled standalone compiler
checks, and native Windows coverage. Compiler regressions belong in
`backend/tests/`; every passing regression prints exactly `PASS\n`.

Architecture and bring-up notes live in [`backend/docs/`](backend/docs/).

## License

Renvo is licensed under the [Apache License 2.0](LICENSE).
