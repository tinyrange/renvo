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
| `freebsd/amd64` | static PIE ELF executable |
| `openbsd/amd64` | static PIE ELF executable |
| `netbsd/amd64` | static ELF executable |
| `wasi/wasm32` | WebAssembly module |
| `browser/wasm32` | Browser HTML containing WebAssembly |
| `vm/vm32` | Deterministic Renvo bytecode (`RNVB`) |

The frontend supports packages and modules, local replacements, build tags and
target-specific files, `//go:embed`, and an offline module cache. Language
coverage includes ordinary control flow, methods, maps, interfaces, closures,
defer/panic/recover, arrays and slices, complex values, goroutines, channels,
`select`, and the builtins needed by Renvo itself. Generics and cgo are
currently out of scope.

Concurrency is a frontend feature: it lowers to the pluggable
`renvo.dev/x/runtime` handler API before the compact backend unit. The bundled
`x/runtime/serial` handler provides cooperative, serialized execution rather
than parallel execution. A target or future microcontroller RTOS can implement
the same handler contract without adding channel operations to a backend.

The bundled `sync` package follows Go's zero-value API for `Mutex`, `RWMutex`,
`WaitGroup`, `Once`, and `Cond`. Blocking operations cooperate with the active
handler, so the same code works with the serialized reference scheduler and a
future target scheduler.

Install a handler before the first concurrency operation. Synchronizing through
a channel drives queued work; `Drain` is available when an event loop wants to
run all currently runnable jobs:

```go
import (
	runtime "renvo.dev/x/runtime"
	"renvo.dev/x/runtime/serial"
)

func main() {
	handler := serial.New()
	runtime.EnableGoroutines(handler)

	values := make(chan int)
	go func() { values <- 42 }()
	println(<-values)
}
```

Packages may also contain `.c` files. The initial C11 frontend is a compact
source adapter into the same package checker, linker, unit format, and backends
used for Go, allowing direct calls between C and Go functions in one package.
Its current scalar/control-flow subset is deliberately smaller than the Go
frontend; see [`internal/c11/README.md`](internal/c11/README.md) for its exact
scope and growth boundary.

## Build and try it

A Go-built bootstrap uses a sibling standalone backend during development.
The `renvo` artifacts themselves are built by Renvo and always contain the
backend in-process:

```sh
go build -o renvo-backend ./backend
go build -tags renvo_bundle -o renvo-bootstrap ./cmd/renvobootstrap

./renvo-bootstrap \
  -t linux/amd64 -o hello ./path/to/hello-package
```

A package can mix the two source languages without cgo:

```go
package main

func main() { print(cAdd(20, 22)) }
```

```c
int cAdd(int left, int right) { return left + right; }
```

The bootstrap looks for `renvo-backend` beside its own executable. Tooling that
keeps the backend elsewhere can pass `-bootstrap-backend <path>` immediately
after the bootstrap executable name.

The C driver can also emit an ordinary Linux/x86_64 ELF relocatable object.
It treats one explicit C file as a standalone translation unit, so no
`go.mod` is required:

```sh
# hello.c contains:
#include <stdio.h>
int main(void) { puts("Hello, world!"); return 0; }

renvo cc -c hello.c -o hello.o
cc hello.o -o hello
./hello
```

The system C driver is used only for startup objects and the final link. Renvo
is the compiler for `hello.c`: it searches installed and `-I`/`-isystem`
headers, retains referenced external declarations from the real header, emits
NUL-terminated C string data and an undefined `puts` with a standard x86_64
PLT relocation, and leaves libc resolution to the system link. The C frontend
also implements the macro and GNU C surface exercised by the pinned Linux
bring-up; unsupported constructs fail explicitly. Its current boundaries are
documented in `internal/c11/README.md`.

Running `renvo` with no arguments or with `--help` prints the complete command
reference and target list.

## Generated and custom backends

The included backend definitions are generated from the closed definitions in
`backend/definitions/` and checked in. A normal `go build ./cmd/renvo` embeds a
fixed native VM32 seed and promotes target-specialized native compilers into a
content-addressed cache on first use. A warm compile executes the cached native
compiler directly. It needs neither the definitions nor a checkout-local
backend at runtime. Native targets share one `native_v1` catalog; Wasm and VM32
use the separate `structured32` family. See
[`backend/definitions/README.md`](backend/definitions/README.md) for the schema
and architecture workflow. After changing an included definition, refresh the
checked-in layers:

```sh
go generate ./backend/definitions
go generate ./internal/targetinfo
go generate ./internal/backendcompiled
```

For distributions that prefer predictable first-run latency over the smallest
portable bootstrap boundary, the native backend can instead be linked into the
same Go executable:

```sh
go build -tags renvo_native_backend -o renvo ./cmd/renvo
```

That variant compiles built-in targets in-process and does not use the native
compiler promotion cache. The untagged build remains the VM32-only portable
frontend; external `-backend` definitions continue to use CompilerJIT so they
do not require regenerating the executable.

A custom definition is prepared by CompilerJIT and executed by the host
command. Passing source prepares it on first use and reuses a content-addressed
cache entry afterward:

```sh
renvo -backend machines.rtg -t acme/amd64 -o app ./cmd/app
```

Preparation can instead produce a host-specific, single-target artifact that
does not require the original definition:

```sh
renvo backend build machines.rtg -t acme/amd64 -o acme-amd64.rtgb
renvo -backend acme-amd64.rtgb -t acme/amd64 -o app ./cmd/app
```

An explicit `-t` remains authoritative when a definition exports several
targets, and must match a prepared artifact. Environments that cannot prepare
or execute a host-native backend, including WASI, use a process-capable Renvo
host for these two commands; that host may still emit `wasi/wasm32` output.

The [M5NanoC6 example](examples/m5nanoc6/README.md) uses this path to combine a
shared RV32IM definition with an ESP32-C6 runtime and flash image. Its emulator
oracle and SDK-free GPIO7 blink provide an end-to-end microcontroller bring-up
without adding the board to the compiled-in host target list.

The `vm/vm32` target uses 32-bit words and pointers and is executed by
`renvo.dev/std/vm`. Callers provide hard instruction and linear-memory limits
and receive deterministic step, peak-memory, output, file, exit, and trap
diagnostics. A 64-bit `vm/vm64` target is planned as later work.

```go
result := vm.Run(program, vm.Limits{
    Steps:  100_000,
    Memory: 32 * 1024,
})
```

`vm.RunConfig` additionally supplies arguments, environment entries, standard
input, and an isolated in-memory filesystem. The VM never reads host state.

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

The bootstrap can also compile and run ordinary Go test functions
with Renvo's `testing` package:

```sh
./renvo-bootstrap test ./path/to/package
```

Test files remain ordinary `_test.go` sources with `func TestXxx(t *testing.T)`
entry points. The command builds a temporary Renvo test main beside the source
package, runs it for the native host target, propagates failure through its
exit status, and removes the generated files afterward.

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
./renvo-bootstrap \
  -tags renvo_bundle -t linux/amd64 -s \
  -o renvo-standalone ./cmd/renvo
```

`renvo-standalone` contains the standard library and an in-process backend. It
can be copied to an empty directory and used without a Go installation,
repository checkout, adjacent data files, or backend process.

Useful development overrides are:

- `RENVO_STDROOT`: standard-library source tree; defaults to the embedded copy
  in bundled builds.
- `RENVO_MODCACHE`: read-only, pre-populated module cache for offline
  dependencies.

Renvo never fetches dependencies while compiling. Use a local `replace`, a
module `vendor` directory, or populate `RENVO_MODCACHE` beforehand.

## Repository layout

```text
cmd/renvo/          command-line compiler
cmd/renvoapk/       Renvo-built Android NativeActivity packager
cmd/renvorepl/      experimental pure-Renvo interactive compiler
cmd/renvoide/       beta graphical development environment
internal/           parser, checker, loader, lowering, linker, and driver
std/                Renvo's target standard library
forms/ and ide/     reusable IDE and UI packages
frontend_tests/     package, diagnostic, self-host, and standalone acceptance tests
backend/            code generators, runtime shell, target descriptions, and backend tests
tools/check         canonical development and PR validation driver
```

The root is the frontend module, published under the canonical path
`renvo.dev`. Backend implementation details are isolated under `backend/`.
The frontend/backend wire format is specified in
[`backend/unit/schema.json`](backend/unit/schema.json) and documented in
[`backend/docs/unit-v1.md`](backend/docs/unit-v1.md).

## Development and testing

The repository contains independent programs in `backend/tests/` and generated
corpus modules in `frontend_tests/`, so `go test ./...` is intentionally not the
whole-project command. Use the checked-in test driver instead:

```sh
./tools/check preflight  # sub-minute development check
./tools/check full       # PR-level backend, gates, corpus, and self-hosting
```

On systemd-based Linux, long checks can be isolated from the rest of the user
session without changing any compiler limit:

```sh
RENVO_CHECK_MEMORY_MAX=4G ./tools/check full
```

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

The GitHub Actions workflow runs the complete backend matrix, compiler resource
and normalized frontend performance gates, self-hosted frontend corpus,
bundled standalone compiler checks, and native Windows coverage. Absolute
runtime and WASI RSS gates remain part of `./tools/check performance`; they are
not used as pass/fail signals on variable shared runners. Compiler regressions
belong in `backend/tests/`; every passing regression prints exactly `PASS\n`.

Randomized differential testing compares deterministic, type-correct programs
under host Go and Renvo. Each seed contains shuffled cycles of arithmetic,
control flow, arrays and slices, structs and methods, interfaces, maps,
strings, closures, defer/recover, complex numbers, multiple results, pointers,
embedding, and short-circuit evaluation. A discrepancy is automatically
reduced while preserving successful host execution and its Renvo failure
class:

```sh
go run ./cmd/renvodiff -seed 1 -count 1000
go run ./cmd/renvodiff -minimize path/to/reproducer.go
```

Findings are written below ignored `sandbox/difftest/` directories with the
original source, minimized source, and both execution results. Long campaigns
should be placed under an external memory limit, for example with
`systemd-run --user --wait --pipe -p MemoryMax=4G` on Linux. The check driver's
memory limit above is equivalent for repository test suites.
The default keeps one independently generated feature case in each program so
one unsupported construct cannot mask unrelated discrepancies. Increase
`-cases` when throughput matters more than isolating each finding.
Use `-minimize-findings=false` for a fast discovery sweep, then pass selected
saved sources back through `-minimize` for focused reduction.
Use `-family expression-tree` (or another named family) to concentrate a
campaign on one language feature; `-list-families` prints the available names.

Architecture and bring-up notes live in [`backend/docs/`](backend/docs/).

## License

Renvo is licensed under the [Apache License 2.0](LICENSE).
