# RFE source packages

An RFE is an editable UTF-8 text archive beginning with `RFE 1`. Sections start
with an entire line of the form `-- filename --`. A delimiter line is reserved
even inside a Go raw string. Version 1 compiles top-level `.go` and `.lower`
sections; other sections may hold accompanying text. Section paths must be
relative and cannot traverse out of the archive. Archives are limited to
32 MiB and 256 sections.

For example, a personality depending on an existing CPU has this shape:

```text
RFE 1

-- rfe.json --
{
  "format": 1,
  "name": "example-user",
  "import": "renvo.dev/emulators/exampleuser",
  "entry": "Main",
  "requires": [{"name": "pdp11"}]
}

-- main.go --
package exampleuser

import "renvo.dev/emulators/pdp11"

func Main(args []string) int {
    cpu := pdp11.CPU{}
    // Attach a bus, load an executable, and dispatch instructions here.
    _ = cpu
    return 0
}
```

`name` identifies the dependency file, `import` identifies its Go package, and
optional `entry` names an executable function `func([]string) int`. Libraries
and CPU packages omit `entry`. Each dependency may include `sha256`, the hex
SHA-256 digest of the complete dependency source file. Formatting changes that
digest, so format before pinning dependencies.

`renvoemu build|test|inspect` resolves dependencies offline from sibling
`name.rfe` files, or the directory selected by `-I`. It rejects missing or
conflicting dependencies, cycles, import identity collisions and digest
mismatches. Extraction rewrites archive import identities into one temporary
module, so the CPU actually comes from the referenced archive. Temporary files
are removed on completion. `-renvo-root` selects the support-runtime checkout;
by default the command finds the current checkout. RFE Go sections are trusted
host code with normal Go privileges, not a sandbox for untrusted extensions.

## Formatting

`go run ./cmd/renvofmt -w emulators` formats embedded Go and lowering DSL with
gofmt, indents and validates the manifest, sorts dependencies and sections, and
normalizes section spacing. The manifest is always first. Without `-w`, a
single file is printed to standard output. `-l` lists changed files and
`-check` fails on invalid or noncanonical files. Directory traversal recognizes
`.rfe` alongside existing `.rtg` and `.rbe` files. Formatting is idempotent and
does not write a file when its validation fails.

## Instruction lowering

A `.lower` section describes pure state transformations using Go-shaped
syntax, with explicit unsigned 64-bit expressions and width truncations:

```go
package examplecpu

//rfe:word 16
//rfe:state 8

//rfe:instruction 0177770 0005200
func increment(opcode uint64) {
    guard(opcode&7 != 7)
    state[opcode&7] = u16(state[opcode&7] + 1)
}
```

This is a syntax example, not a full PDP-11 INC definition: actual INC also
updates flags. See `pdp11.rfe` for complete rules. The annotations define opcode
width (1–32 bits), state words (1–256), and mask/value matching. Rules may use
initial decode guards, local assignments, state reads/writes, arithmetic,
bitwise operations, comparisons, constant shifts, `choose`, and `u8`/`u16`/`u32`
truncation. Loops and arbitrary calls are rejected. Decoder validation rejects
overlapping masks and checks state indices over the rule's accepted encodings;
each rule may have at most 16 variable opcode bits.

Generated `Decodable` and `Lower` functions feed a small, architecture-neutral
SSA IR. Constant folding, mask propagation and dead-value elimination remove
overwritten intermediate flags. State stores commit at the end of the block.
Renvo emits call-free native state transformations from this IR using its
existing amd64 and arm64 code generators. Memory, devices, traps and syscalls
remain in the architectural Go implementation and end a pure block.

The initial PDP-11 engine lowers selected register instructions, excluding PC
operands. It compiles after eight visits, limits blocks to 32 instructions and
the cache to 4096 entries with an 8 MiB native arena. Cache exhaustion falls back
to interpretation. Every dispatch revalidates instruction bytes through a
side-effect-free bus peek; changed bytes invalidate the block. MMU checks and
accessed bits, trace traps and instruction budgets remain architectural. This
is intentionally a partial lowering engine, not a claim that all guest code
executes natively.

## Device contracts and provenance

`internal/rfe/runtime` defines byte/word bus access, execute/read/write intent,
structured unmapped/boundary/permission/alignment/device faults, and safe
unobserved peeks. Its event queue orders events by tick and then insertion
sequence. These contracts are adapted from the Rust sister project
[`tinyrange/renvo_emu`](https://github.com/tinyrange/renvo_emu), revision
`6b0c7221ca32e83903797d2c86a8effdcd62550a`, especially
`crates/remu-core/src/bus.rs` and `event.rs`. This is a Go implementation of
those semantics, not binary compatibility or a Rust dependency.

The full-system RFE supplies console interrupts, a KW11-style clock, and an
RK11 controller with in-memory RK05 media and DMA. The user RFE instead supplies
split instruction/data memory and handles syscall traps. Both use the same CPU
archive and engine. Additional CPUs and personalities can use the same package
resolution, bus contracts, IR and native block interface without PDP-11 logic
in the compiler.

The CPU is adapted from `examples/pdp11/cpu.c`, based on Paul Nankervis's
`pdp11-js` revision `605cc23eada3d831be54e537fe94307f1b80a85b`, with attribution
preserved in the source. FP11 instruction formats, precision, status bits and
exceptions follow chapter 6 of DEC's
[PDP-11 Architecture Handbook](https://www.bitsavers.org/pdf/dec/pdp11/handbooks/EB-23657-18_PDP-11_Architecture_Handbook_1983.pdf).
The implementation uses exact integer/rational intermediates, explicit DEC
quantization, three alignment guard bits, and a 59-bit D-mode MOD product.
The [SIMH FP11 implementation](https://github.com/simh/simh/blob/master/PDP11/pdp11_fp.c)
was also consulted for addressing and exception behavior. Floating registers
belong to the shared CPU, so fork copies them and exec initializes new state.

V7 ABI behavior follows the historical kernel's `sysent.c`, `trap.c`,
[signal handling](https://www.tuhs.org/cgi-bin/utree.pl?file=V7/usr/sys/sys/sig.c),
[signal syscalls](https://www.tuhs.org/cgi-bin/utree.pl?file=V7/usr/sys/sys/sys4.c)
and filesystem structures. Signal dispositions are reset on delivery except
SIGILL and SIGTRAP. Pending bits coalesce, SIGKILL cannot be caught or ignored,
fork inherits dispositions but clears pending signals and alarms, and exec
preserves ignored dispositions and the alarm while resetting handlers.
Only the scheduler mutates process state; terminal notifications and completed
host reads arrive over channels. A descriptor retains an in-flight host read
across EINTR, preventing the interrupted operation from consuming input meant
for a subsequent read. Generic Go readers cannot be forcibly cancelled; callers
embedding the kernel own the lifetime and closing of their input readers.

The initial user environment
limits its in-memory filesystem to 64 MiB, live processes to 64, descriptors to
20 per process and each pipe to 4096 bytes. It supplies a useful historical
shell environment, with remaining compatibility limitations listed in the
[emulator README](../emulators/README.md).
