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

### Backend support tiers

Backend tiers describe the testing and maintenance contract for a target. They
are not a ranking of usefulness, and they do not imply language or API
compatibility beyond the tests named by the tier. A target can share code with
a higher tier without inheriting that tier: promotion requires its own required
coverage.

- **Tier 1 — gated.** The target runs the complete applicable backend regression
  and end-to-end suites in required CI and has hard compiler performance or
  resource limits. A functional or performance regression blocks the merge
  queue.
- **Tier 2 — supported.** The target is supported on mainline and required CI
  compiles and executes a target binary, either natively or in an emulator. Its
  functional coverage may be smaller than the Tier 1 suite, and it has no
  compiler performance or resource regression gate. Failures still block the
  merge queue. Compilation or image inspection alone is not enough for Tier 2.
- **Tier 3 — experimental.** Confidence comes from manual smoke testing on the
  target. Generic definition parsing or generation checks may run in CI, but
  CI does not make a target-specific functional guarantee and does not replace
  testing on real hardware or the destination system.

The current classification is:

| Tier | Targets |
| --- | --- |
| **Tier 1** | `linux/amd64`, `linux/386`, `linux/aarch64`, `linux/arm`, `windows/amd64`, `windows/386`, `darwin/arm64`, `wasi/wasm32`, `vm/vm32` |
| **Tier 2** | `linux-object/amd64`, `llvm/linux-amd64`, `esp32c6/riscv32`, `esp32c6-jtag/riscv32`, `esp32s3/xtensa_lx7` |
| **Tier 3** | `windows/arm64`, `browser/wasm32`, `freebsd/amd64`, `openbsd/amd64`, `netbsd/amd64`, `linux-kernel/amd64`, `c89/hosted32`, `c89/hosted32-auto`, `c89/freestanding32`, `android/arm64`, `ios/arm64`, `esp32p4/riscv32`, `msdos/8086`, `msdos/8086-mz`, `uefi/amd64` |

New backends start in Tier 3. Promotion to Tier 2 requires a required CI test
that executes generated code for that target and a commitment to keep the
target working on mainline. Promotion to Tier 1 additionally requires the full
applicable suites and explicit performance or resource budgets. If required
coverage can no longer be maintained, the target should be moved to the lower
tier in the same change rather than silently weakening its tests.

### Performance gates

Performance limits are regression budgets, not benchmark claims. The compiler
tests make three attempts and use the best observed elapsed time and peak RSS;
binary-size limits apply to the stripped compiler artifact. The current hard
limits are:

| Gate | Targets | Required limits |
| --- | --- | --- |
| Fixed-target compiler resources | `linux/amd64`, `linux/386`, `linux/aarch64`, `linux/arm`, `windows/amd64`, `windows/386` | 16 MiB peak RSS; 320 KiB compiler |
| Fixed-target compiler elapsed time | The same Linux and Windows targets | 50 ms; run by `./tools/check performance` outside shared CI |
| Darwin fixed-target compiler | `darwin/arm64` | 175 ms; 640 KiB compiler; run on the native macOS CI runner |
| WASI fixed-target compiler | `wasi/wasm32` | 150 ms; 20 MiB peak RSS; 384 KiB compiler |
| Self-hosted frontend | Native Linux target for the runner | 42 MiB peak RSS; 4 MiB stripped stage-3 compiler |
| Self-hosted VM backend | `vm/vm32` | 2 MiB compiler bytecode; 400,000 VM steps; 80 MiB peak VM memory |
| Self-hosted VM frontend | `vm/vm32` producing `linux/amd64` | 6 MiB frontend bytecode; 4 MiB output compiler; 12 billion VM steps; 150 MiB peak VM memory |

`./tools/check ci-performance` runs the resource and binary-size gates that are
stable on shared Linux runners, together with the WASI gate. It also records
normalized frontend CPU time as telemetry, but CPU time is not currently a
hard frontend limit. `./tools/check performance` adds the 50 ms fixed-target
elapsed-time gate. The native Darwin job and the required backend/frontend jobs
own the Darwin and VM gates respectively. The VM backend corpus additionally
caps each regression program at 500 million steps and 16 MiB of VM memory.

The frontend supports packages and modules, local replacements, build tags and
target-specific files, `//go:embed`, and an offline module cache. Language
coverage includes ordinary control flow, methods, maps, interfaces, closures,
defer/panic/recover, arrays and slices, complex values, goroutines, channels,
`select`, cgo-style explicit C package boundaries, and the builtins needed by
Renvo itself. Generics remain out of scope.

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
used for Go. Mixed packages use `import "C"` for Go-to-C calls and `//export`
for the functions C may call back into.
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

A package can mix the two source languages through an explicit cgo-style
boundary without invoking a host C toolchain:

```go
package main

import "C"

//export goValue
func goValue() int { return 40 }

func main() { print(C.cAdd(1, 1)) }
```

```c
extern int goValue(void);
int cAdd(int left, int right) { return goValue() + left + right; }
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

Without `-c`, `renvo cc` builds a complete executable and needs neither a
`go.mod` nor a host C toolchain:

```sh
renvo cc hello.c -o hello
./hello
```

Executable mode uses Renvo's embedded C headers and demand-selects the matching
library implementations from the same bundle as the Go standard library. The
minimal hosted surface includes the fundamental integer/size/boolean headers,
`stdarg`, memory and string operations, allocation and conversions, character
classification, assertions, streams, and formatted output. Both `main(void)`
and `main(int, char **)` are supported. C identifiers do not fall through to
Go builtins; for example, `print` is not a C output function—include
`<stdio.h>` and use `printf`, `puts`, or the stream APIs.

A native C entrypoint can opt into a narrow Go adapter from the same directory:

```c
#pragma go "board.go"
extern void board_set_led(int on);
```

Only named `.go` files are added to an explicit `cc` build. Their ordinary Go
imports are resolved transitively, letting C application code use small typed
adapters for Renvo board and device packages without implicitly compiling every
Go file beside the C source.

Microcontroller applications use one target-selected board package:

```go
import "renvo.dev/device/board"

func main() {
	board.RGB.Set(16, 0, 0)
}
```

The Web IDE offers board choices such as `m5nanoc6/riscv32`,
`m5atoms3lite/xtensa_lx7`, and `m5tab5/riscv32`. Each choice maps to a shared
chip backend plus one board build tag. From the CLI, express the same selection
as `-t esp32c6/riscv32 -tags m5nanoc6`. Only capabilities present on that board
are declared, so using an unavailable device is a compile-time error.
Supported-board metadata and demo membership live in
`device/board/catalog.json`, not in the RTG machine definitions.

Running `renvo` with no arguments or with `--help` prints the complete command
reference and target list.

## Generated and custom backends

The included backends are generated from the closed definitions in
`backend/definitions/`, checked in, and compiled into ordinary Go builds. A
normal `go build ./cmd/renvo` therefore needs neither the definitions nor a
checkout-local backend at runtime. Native targets share one `native_v1`
catalog; Wasm and VM32 use the separate `structured32` family. See
[`backend/definitions/README.md`](backend/definitions/README.md) for the schema
and architecture workflow. After changing an included definition, refresh the
checked-in layers:

```sh
go generate ./backend/definitions
go generate ./internal/targetinfo
go generate ./internal/backendcompiled
```

A custom definition is prepared with the compiled-in host backend and executed
by the host command. Passing source prepares it on first use and reuses a
content-addressed cache entry afterward:

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

The Tier 3 `uefi/amd64` definition emits a PE32+ EFI application without a
Windows import table. `renvo.dev/device/uefi` exposes the firmware system table,
text console, boot and runtime services, GOP framebuffer, and Simple File
System protocols. The web IDE includes hello, graphics, filesystem, and Linux
bootloader demos and downloads them as `BOOTX64.EFI`. Local OVMF smoke tests
boot images in QEMU and exercise console output, timing, protocol discovery,
filesystem access, and a complete Linux handoff:

```sh
./tools/uefi/test-qemu
./tools/uefi/test-linux-qemu
```

The Linux smoke test downloads an official Alpine `vmlinuz-virt` and verified
minirootfs, builds a tiny initramfs around Alpine BusyBox, then checks for a
userspace success marker over QEMU's serial port. The bootloader uses Linux's
native x86-64 boot protocol rather than the kernel EFI stub.

Set `RENVO_OVMF_CODE`, `RENVO_UEFI_QEMU`, or `RENVO` when those tools or the
compiler are outside their usual locations. This test is deliberately local;
it is not a Tier 2 CI guarantee.

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
internal/           parser, checker, loader, lowering, linker, and driver
std/                Renvo's target standard library
forms/              reusable UI packages
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
