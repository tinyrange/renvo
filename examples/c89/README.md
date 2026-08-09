# C89 CompilerJIT backend

`c89.rtg` is a custom backend definition, not a frontend transpiler. It
implements `direct_emitter_v1`, is prepared through the same CompilerJIT path
as the ESP32 targets, consumes the canonical linked unit, and emits an ISO C89
translation unit.

All current targets use the `c89vm32` RTG architecture: little-endian,
eight-bit bytes, 32-bit Renvo `int` and pointers, four-byte stack words, and
the internal register-machine ABI declared in `c89.rtg`. These properties are
fixed before C emission; the generated C never infers target endianness or
layout from its compiler.

Three profiles are available:

- `c89/hosted32` explicitly requires 32-bit C `unsigned int` and pointers,
  emits `main`, and implements Renvo process arguments, selected environment
  variables, and filesystem operations with C89 stdio.
- `c89/hosted32-auto` keeps the same Renvo machine contract but automatically
  selects an exact 32-bit unsigned carrier from `UINT_MAX` or `ULONG_MAX`.
- `c89/freestanding32` uses the explicit carrier contract, emits the external
  six-character entry point `rgmain`, and has no hosted startup or libc
  references.

Named C89 negative-array assertions reject incompatible `CHAR_BIT`, integer,
and pointer widths. The runtime represents Renvo signed values as unsigned
bit patterns, including overflow, division, remainder, comparisons, narrowing,
unaligned memory, and shifts. Direct and indirect calls use bytecode program
counters rather than C function pointers, so they do not depend on converting
between C object and function pointer representations. Calls emitted through
the internal ABI use the indirect path in normal generated programs, while
retaining the same register and overflow-argument convention.

The hosted profile maps `open`, `close`, `read`, `write`, `read_at`, and
`write_at` to the ISO C stream API. It exposes `PATH`, `PWD`, `RENVO_STDROOT`,
`RENVO_MODCACHE`, and `TMPDIR` when they exist. ISO C89 has no portable API for
Unix executable permission bits, so `chmod` validates an open descriptor but
does not change host permissions; bootstrap scripts apply executable mode
after the C-generated compiler writes an ELF file.

Generate C with:

```sh
renvo \
  -backend examples/c89/c89.rtg \
  -t c89/hosted32 \
  -s \
  -o program.c \
  path/to/main.go
```

Each compiler has a separate, digest-pinned Docker base. The matrix currently
covers GCC 12.2, Clang 14, TCC 0.9.27, PCC 1.2.0.DEVEL, and the historical GCC
4.9.2 from Debian Jessie. It compiles and executes both hosted profiles, then
compiles freestanding output and verifies that its object has no undefined
symbols. Modern GCC and Clang also compile the explicit profile with a
deliberately incompatible 64-bit pointer model and require the diagnostic to
name `renvo_assumption_pointer_width`.

The matrix consumes one generated file for each profile, a hosted write
fixture, and a directory of generated corpus programs:

```sh
./tools/c89/matrix \
  hosted-explicit.c hosted-auto.c freestanding-explicit.c \
  hosted-write.c hosted-fault.c corpus/
```

The integration test builds corpus inputs already accepted by native backends
for signed division/remainder, overlapping slice expansion, function values
and indirect calls, 64-bit operations on a 32-bit target, unsafe indexed
pointers, variadic calls, and scratch-register decrement aliasing. Every one is
compiled and executed by every compiler in the matrix. It also verifies that
an uncaught runtime fault prints one `panic` line and exits with status 2.

The normal Go test validates the CompilerJIT integration and source-level C89
rules without requiring Docker. The full container matrix is opt-in:

```sh
RENVO_C89_DOCKER=1 go test ./internal/backendjit -run TestCompilerJITC89DockerMatrix
```

The deeper bootstrap gate uses a 128 MiB arena for the C89-hosted compiler. For
each of GCC, GCC 4.9, Clang, TCC, and PCC it performs this complete chain:

```text
backend source -> C89 translation unit -> C compiler -> Renvo backend
Renvo frontend unit -> C89-built backend -> stage 1 frontend
stage 1 frontend -> stage 2 frontend -> byte-identical stage 3 frontend
```

The C89 process remains a 32-bit program, but the self-hosted backend and
frontend are emitted as `linux/amd64`, exercising Renvo's retargeting boundary
and giving the frontend its normal 128 MiB hosted arena. Run the gate with:

```sh
RENVO_C89_BOOTSTRAP_DOCKER=1 \
  go test ./internal/backendjit -run TestCompilerJITC89DockerBootstrap -timeout 45m
```

The generated-source test rejects post-C89 constructs including C99 comments,
mixed-width convenience headers, `inline`, `long long`, and C11 static
assertions. Every compiler execution in this matrix happens inside its own
Docker image.
