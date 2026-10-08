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

Expressions are parsed once into a typed tree shared by validation and code
generation. Guards and state indices must depend only on the opcode; values
are unsigned integers or booleans with checked operand types. Generated
`Execute` and `Lower` functions use the same validated decoder rules and instruction
semantics. The PDP-11 interpreter uses `Execute` for ALU operations after
resolving operands, including memory operands and byte operations. `Lower`
feeds a small, architecture-neutral SSA IR. Grouped dispatch is emitted directly
into each entry point to avoid a second runtime dispatch. Constant folding,
mask propagation and dead-value elimination remove
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

## AArch64 user-mode extension

`aarch64.rfe` and `linux-arm64-user.rfe` add an initial little-endian A64 integer
CPU and static-ELF Linux user personality. A normalized decoder directly feeds
the shared pure IR without relaxing the `.lower` decoder's 16-variable-bit
limit. Hot pure blocks progress from interpretation to IR to native execution.
Native mode also compiles guarded scalar memory blocks for a compatible
single-threaded memory implementation. Terminal branches compute successors in
compiled code; a bounded native dispatcher links already validated hot entries
within one unchanged executable context. Misses, faults, executable stores,
pairs/literals, division and traps still return to architectural dispatch.
The shared builder now value-numbers immutable expressions while preserving
architectural state versions, and supports modular multiplication through the
existing native emitters. Extended integer operations and conditional
select/compare lower into pure blocks. The Linux personality uses address-space
identity and code-page generations for hot entry guards; custom unversioned
memories retain byte-by-byte code revalidation.

A pinned CoreMark port now passes both seed sets in every tier. Raw diagnostic
timings expose a large remaining QEMU gap; no paired scoring win is established.
This implements bounded loop traces, not the general optimizing CFG engine. See [the detailed design](aarch64-user.md) for implemented coverage,
limits, the effect-aware optimizer plan, acceptance tests and benchmark rules.

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
host I/O arrives over channels. Descriptor installation and removal share one
ownership path across close, dup and fork. A syscall table keeps ABI argument
layouts, trace names and named handlers together. The kernel retains one shared
terminal read across EINTR, close and reopen, so all terminal descriptors consume
the same buffered input without competing host reads. Host writes use copied buffers and run asynchronously,
serialized within each kernel, so blocked output cannot prevent signal delivery.
Generic Go readers and writers cannot be forcibly cancelled: a host write may
finish after the guest receives EINTR. Callers embedding the kernel own the
lifetime and closing of their host streams.

Because this personality has no guest init process, the kernel adopts children
whose parent exits and automatically reaps them, including children which have
already exited. Direct children of a live process retain their normal wait status.

Floating arithmetic deliberately retains exact rational intermediates. The
embedded `BenchmarkFloating` measures F/D add, multiply and divide separately;
an Apple M4 baseline measured 376–540 ns and 31–44 allocations per instruction.
This identifies an optimization opportunity without trading away D precision.

Profiling the shared ALU found decoder dispatch overhead and redundant integer
mask operations. Direct grouped dispatch, single-bit selection and modular
truncation simplification retain one authored set of instruction semantics.
On Apple M4, three isolated one-second runs of `BenchmarkRegisterBlock` gave
these medians against commit `d75f89bb` (each iteration executes 16 increments):

| Mode | Before | After |
| --- | ---: | ---: |
| Interpreter | 276.4 ns | 262.6 ns |
| IR | 361.2 ns | 247.1 ns |
| Native | 94.39 ns | 77.65 ns |

Native code shrank from 2,052 to 1,292 bytes, and IR allocation from 1,024 to
640 bytes per iteration. These are microbenchmark results, not whole-program
speedups. Interpreter execution still trails the earlier handwritten ALU.

The initial user environment
limits its in-memory filesystem to 64 MiB, live processes to 64, descriptors to
20 per process and each pipe to 4096 bytes. It supplies a useful historical
shell environment, with remaining compatibility limitations listed in the
[emulator README](../emulators/README.md).


### Native emission and profiling

Pure and effect-aware scalar blocks use bounded SSA register allocation with
constant rematerialization and spill fallback. Variable shifts/rotates carry an
explicit 32/64-bit width; counts are masked to that width. Conditional select
uses a backward SSA reference for its third operand and lowers to a native
conditional move without suppressing already-ordered memory effects. Guest
architectural state is still committed at block boundaries.

The native link dispatcher keeps its working state in preserved registers across
leaf calls and retains all entry, budget, ownership and partial-progress guards.
Legacy assembly calls retain a 64-instruction ceiling; supported Linux/amd64+cgo
foreign sessions have an independently configured ceiling of up to 65,536. Linux emulator `-stats` additionally enables native
perf-map symbols. Map files are exclusively created as `/tmp/perf-PID.map` with
mode 0600 and retained after exit for report symbolization; failures are nonfatal
and reported as `profile_errors`. Other hosts retain ordinary stats without this
Linux profiling facility. See `aarch64-validation.md` for unchanged-guest timing
results, validation scope and remaining limitations.


Arithmetic/logical flags now have explicit width-aware IR operations, with
separate direct condition predicates when their operands are available. Native
entries calculate their own host flags rather than consuming ambient flags from
an unrelated operation. Architectural NZCV remains reconstructible at every
exit. Leaf entries retain the architectural state pointer in a reserved register.
For effect blocks, ordered architectural checkpoints remain intact, while native
retirement/fault-address bookkeeping is emitted on successful return and precise
slow exits rather than every fast access. These changes do not increase the
native dispatch quantum. Bounded loop regions are described below.


AArch64 linked Run entries support real bounded native loops and cyclic traces
through already-validated hot blocks. Loop phis keep forwarded guest values in
registers/private spills; native cold exit maps restore architectural state.
Internal branch exits can continue native dispatch without Go state-management
callbacks. Cold blocks remain at most 16 instructions, while optimized regions
can independently contain up to 256. Supported foreign sessions retire at most
65,536 instructions; unsupported/aliasing cases retain <=64 calls. Step and short
budgets retain original leaf entries,
and dependency checks cover all constituent code pages.

Rich high multiply, leading counts, reversals and explicit carry/NZCV records
survive until host selection. Ordinary SSA operations use allocated registers
directly. Native loops reuse call-local checked page translations. Proven
invariant addresses keep an exact-access fact after successful range/permission
guards, but memory data is always accessed afresh and every store checks clock
overflow. Exit-only flags/selects are reconstructed natively with live operands.
Private immutable arena keys move admission to cold publication, while each
native selection still validates the public count and exact state shape.
This remains bounded trace execution, not a general CFG optimizer or zero-Go
runtime; cold compilation, mapping, traps and quantum scheduling remain in Go.
Linked bodies borrow a bounded dispatcher frame and stable state/context bases;
standalone APIs retain compatibility entries. See `rfe-native-abi.md` for the
shared-frame contract, legacy interoperability and limits.
Guarded region edges refine forwarded PC only after their precise checkpoint
and successful target guard. Prepared dispatcher handles move immutable entry
validation off the quantum path while retaining arena serialization and native
leaf guards. See the batch 10 evidence in `aarch64-validation.md` for CRCs, tests, raw timings
and the still-unmet QEMU performance gate.


The Linux AArch64 process uses `RunQuanta` with independently configured
optimized-region and foreign-session ceilings. `-region-instructions` defaults
to 256 and `-native-instructions` to 65,536. The conservative path can still
amortize serialization across sixteen separate <=64 native calls. Cold work and
architectural slow paths stay outside the lock. Native exit predicates use directly produced flags and
bounded cold SSA maps, and read/write page proofs are separate and call-local.
The fixed 1000-iteration CoreMark diagnostic now has a controlled 0.97-second
median; this does not satisfy CoreMark's score-duration requirement.
