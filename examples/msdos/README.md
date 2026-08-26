# MS-DOS COM custom-backend demo

This example is a small, complete custom backend for 8086-compatible MS-DOS
`.COM` programs. Like [`examples/pdp11v7`](../pdp11v7), the compiler prepares
the target definition on demand and emits an original executable from the Go
source. The result is a headerless COM image loaded by DOS at offset `0x100`;
it is not an emulator container or a prebuilt byte sequence.

The target deliberately uses a 16-bit `int` and pointer model. Its scope is
small programs whose code, initialized data, zeroed data, heap, and stack all
fit in one 64 KiB DOS segment. The backend provides the 8086 scalar,
control-flow, call, memory-copy, and unsigned-division operations used by
compact Renvo programs. Its DOS `INT 21h` runtime implements open/create,
close, read, write, positional I/O, chmod compatibility, process arguments,
environment, and exit. It selects a 24 KiB arena by default, zeroes BSS at
startup, leaves zero-filled BSS out of the file image, and reserves 4 KiB for
the machine stack.

The program also demonstrates project-local assembly. `main.go` declares the
bodyless `rtgasmAnswer` function and `answer_msdos_i8086.rtgasm` implements it
with the target's existing emitter helpers, including a local label and patched
jump. CompilerJIT evaluates that small source fragment in VM32 and inserts the
result at the ordinary function label; no opcode knowledge is added to the
frontend or shared compiler.

From the repository root:

```sh
go run ./cmd/renvo \
  -backend examples/msdos/msdos_com.rtg \
  -t msdos/8086 -s -o sandbox/hello.com \
  examples/msdos
```

The browser bundle lists `examples/msdos` beside the PDP-11 example. Opening it
loads this RTG definition, prepares the `msdos/8086` compiler as a cached VM32
backend, and downloads the result as `app.com`. The same prepared target accepts
both Go projects and C projects created in the browser.

Run `sandbox/hello.com` under DOSBox, real MS-DOS, or an 8086 emulator with a
COM loader and the required `INT 21h` file/process functions. It has also been exercised
with [Vbitz's emulator](https://gist.github.com/Vbitz/2d92f7ccea1fe257bf53baff97072069)
through its `RunCOMFile` entrypoint. The expected output is:

```text
Hello from Renvo on MS-DOS!
```

## Corpus qualification

`renvodosqualify` compiles every positive backend program and every quick,
extended, and regression frontend module for `msdos/8086`, executes each COM
image, and compares its output with the checked-in expectation:

```sh
go run ./cmd/renvodosqualify -runner /path/to/comrun
```

The runner is intentionally external: it must accept a COM path as its first
argument, forward emulated DOS stdout and stderr to its own corresponding
streams, and exit nonzero for an emulator fault or nonzero DOS exit. A small
wrapper around the linked emulator's `RunCOMFile` function satisfies that
contract. The emulator's DOS create implementation must return a read/write
handle, matching `INT 21h/AH=3ch`.

Use `-suite backend` or `-suite frontend` for one corpus, and `-filter REGEXP`
for a focused rerun. The command runs every selected case even after a failure.
It starts with the target's 24 KiB arena and retries progressively smaller
arenas when code and static data need more of the segment. Passing programs are
executed and compared with their checked-in expectations. Cases that inherently
require a wider Go data model, a foreign runtime operation, or more than one
COM segment are reported as `target-model`, `target-platform`, or
`target-memory`; other compile, emulator, and output failures still make the
qualification command fail.

Prepared native backend execution is currently process-global, so the command
requires `-jobs 1`. A full run takes several minutes and is deliberately not
part of normal preflight.

The generated file contains only 8086 instructions; it does not use Renvo's
separate `-code16` mode. That mode implements GCC-style i386 ILP32 code with
operand- and address-size prefixes and therefore requires a 386-aware runtime.
