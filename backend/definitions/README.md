# Backend definitions

Renvo has two backend families:

- `native_v1` describes register machines that emit native code, relocations,
  ABI calls, hosted runtime operations, and native images.
- `structured32` contains the existing Wasm and VM32 implementation. It is
  deliberately separate because structured control flow, module index spaces,
  and validation are not native-machine concepts.

Each built-in native target is an independent definition entrypoint. Shared
ISA and format fragments live beside those entrypoints and are imported only
by targets that need them:

```text
backend/definitions/
├── x86_64.rtg                 shared ISA and encoder
├── x86_32.rtg
├── aarch64.rtg
├── arm.rtg
├── riscv32.rtg               shared RV32IM used by external board targets
├── elf_amd64.rtg              shared AMD64 ELF formats
├── *_algorithms.rtg            closed shared-ISA generation roots
├── linux_amd64.rtg            complete target entrypoint
├── windows_amd64.rtg
├── linux_kernel_amd64.rtg
├── linux_386.rtg
├── windows_386.rtg
├── linux_aarch64.rtg
├── darwin_aarch64.rtg
├── windows_aarch64.rtg
└── linux_arm.rtg
```

ISA fragments own reusable machine facts, encoders, operation bindings, and
calling conventions shared by more than one target. Target entrypoints own
their unique ABI, runtime boundary, executable/object format, entry sequence,
hooks, and final target declaration. `extend arch` adds target-local bounded
sequences to an imported ISA without duplicating it. Its body inherits the
ISA's virtual-package symbols, so target code uses short names such as
`asmMovRegImm`; generated qualification remains automatic and collision-free.
The Wasm/VM family remains in [`wasm32.rtg`](wasm32.rtg).

External targets may instead use a single `.rbe` file. An RBE begins with an
ordinary RTG definition and appends one or more
`@stdlib "package/file.go"` ... `@endstdlib` sections. Those files overlay the
standard library only for builds selecting that backend, and are retained in a
prepared `.rtgb` artifact. This is the appropriate form when a new operating
system needs both machine/runtime definitions and target-specific Go APIs. See
[`examples/pdp11v7/pdp11_v7.rbe`](../../examples/pdp11v7/pdp11_v7.rbe) for a
complete PDP-11 Unix V7 target and syscall package.

Generated Go is checked in so an ordinary `go build` needs no generator:

```text
go generate ./backend/definitions
go generate ./internal/targetinfo
go generate ./internal/backendcompiled
```

The first command emits each production architecture projection from its
matching `*_algorithms.rtg` root and the authoritative Linux/amd64 production
target projection from `linux_amd64.rtg`. Architecture roots import only their
shared ISA fragment; target projections use a separate built-in namespace and
call those shared algorithms directly rather than using the prepared-backend
adapter. The second command resolves target descriptors and writes the registry
and the non-enforcing source-volume report. Architecture-only roots are
validated by generation but intentionally contribute no registry target. The
third refreshes the ordinary-Go compiled backend and the source bundle used to
prepare external definitions.

## Adding a native architecture

Add a new shared ISA fragment and one self-contained entrypoint per supported
OS/environment directly under `backend/definitions/`; do not add an
architecture switch to the RTG parser or generator.

1. Declare the physical registers once with `registers`.
2. Use `register_class` for overlapping constrained subsets.
3. Use `register_group` for multi-register values such as register pairs.
4. Declare non-flat storage with one or more `address_space` blocks.
5. Map compiler roles in `locations`; names such as `primary` and `stack` are
   roles, not required physical register spellings.
6. Describe recurring encodings in `forms` and instruction variants as rows in
   `instructions`. Fixed-width 16- and 32-bit words use typed `word16` and
   `word32` forms; `address_base` covers the common simple base-register
   operand without requiring a Go hook.
7. Use bounded `sequences` for straight-line encoder, ABI, entry, and runtime
   composition. Calls use ordinary `helper(out, ...)` syntax; declared
   registers, locations, conditions, instructions, and file-local helpers can
   be referenced by their short names in both sequences and embedded Go. Go
   parameters and local variables shadow architecture facts normally.
   Sequences permit typed calls and local values but deliberately have no
   general branch or loop construct.
8. Bind every `direct_emitter_v1` operation to an instruction, bounded
   sequence, or typed Go hook, or explicitly reject an operation supported by
   a later contract.
9. Select a named label-relocation encoding and describe executable/object
   relocation facts in the format declaration.
10. Keep only genuinely shared calling conventions in the ISA fragment. Put
    each target-specific ABI, runtime, format, entry sequence, hook, and final
    target composition in a target-named entrypoint such as
    `linux_riscv64.rtg`; imported ISA symbols are available by short name
    inside its `extend arch` block.
11. Export only the architecture algorithms used by the checked-in compiler.
12. Add the architecture projection to `generate.go`, regenerate, and run the
    complete backend and frontend suites under the repository memory cap.

Embedded Go is an opaque, typed escape hatch. It is appropriate for irregular
encoders, immediate synthesis, branch relaxation, ABI edges, and runtime or
format algorithms with genuinely target-specific control flow. Ordinary label
relocations, ELF/PE image patching, Linux-module ELF construction, entry byte
templates, runtime operation selection, and straight-line ABI emission are
already bounded declarations and should not be reintroduced as hooks. Renvo
checks hook signatures but does not attempt complexity or termination
analysis. Before adding a hook, check whether a table row, an existing form, a
bounded sequence, or a shared format constructor states the difference more
directly.

`TestNativeDefinitionEmbeddedGoMetrics` deduplicates shared declarations across
every native entrypoint and reports semantic Go bytes and declaration counts as
architecture-maintainability evidence. It is intentionally not a numeric
acceptance gate and is not permission to move target algorithms into an
unmeasured file. Reusable generator code may implement a generic format
constructor; machine and OS differences must remain visible as typed `.rtg`
facts.

## Identity and pruning

An entrypoint hash identifies its complete expanded declaration graph,
including composed bounded sequences, and appears in generated provenance
headers. Import directories, comments, and formatting are not semantic. An
imported file's basename is semantic because it supplies the virtual package
used to resolve private helper names. A target semantic identity hashes only
the selected target and its transitive machine, ABI, runtime, format, reachable
bounded-sequence, and embedded Go dependencies. RTGU bindings, target
descriptors, prepared artifacts, and cache keys use the target identity.

Comments, formatting, declaration order, and unreachable architectures do not
change a target identity. Changing a reachable hook or declaration does.
Fixed and prepared generation starts at one selected target and emits only
reachable Go declarations.

## Imports

`@import "relative/path.rtg"` includes another definition fragment at the top
level. Paths are resolved relative to the importing file, nested imports are
supported, and cycles are rejected. Imported files are fragments: the selected
entrypoint owns `definition`, `unit`, and `implements`, while the parser
validates the expanded graph as one closed definition. Every file forms a
virtual package named after its basename: Go helpers and bounded sequences are
short and local by default. `extend arch` inherits the imported ISA's public
statement and helper symbols into the target file while keeping newly declared
helpers target-local. Explicit `package_name.helper` references remain
available when needed. Machine declarations and explicit export names remain
global. Diagnostics retain the filename and position of the fragment that owns
the invalid declaration.

## Schema design checks

`internal/rtg/testdata/native_8086.rtg` and `native_avr.rtg` are schema fixtures,
not shipping backends. They keep the native model honest about:

- 8- and 16-bit words;
- a 20-bit segmented address space;
- Harvard code and data spaces;
- overlapping register subsets;
- multi-register values;
- short branches, restricted immediates, calls, and relocations.

Do not add board policy, macros, arbitrary compile-time evaluation, or a second
instruction IR to satisfy these fixtures.

## Verification

Use focused definition checks while editing:

```text
go test ./internal/rtg ./internal/rtgb ./internal/targetinfo
go test ./internal/backendjit ./internal/backendcompiled
```

Then run the unchanged performance and self-hosting suites:

```text
systemd-run --user --scope \
  -p MemoryMax=4G -p MemorySwapMax=0 \
  -- go test ./backend -count=1

systemd-run --user --scope \
  -p MemoryMax=4G -p MemorySwapMax=0 \
  -- go test ./frontend_tests -count=1
```

The per-target metrics in `backend/docs/machine-definitions.generated.md` are
review aids. The deduplicated native embedded-Go metric is reported by the RTG
tests without a numeric rejection threshold. Compiler and output performance
acceptance remains exclusively defined by the existing hard gates in
`backend/main_test.go`.
