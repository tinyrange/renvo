# Backend definitions

Renvo has two backend families:

- `native_v1` describes register machines that emit native code, relocations,
  ABI calls, hosted runtime operations, and native images.
- `structured32` contains the existing Wasm and VM32 implementation. It is
  deliberately separate because structured control flow, module index spaces,
  and validation are not native-machine concepts.

The built-in native source of truth is the closed catalog rooted at
[`native.rtg`](native.rtg). That small entry point imports one
architecture-owned slice for each native ISA:

```text
native.rtg
└── native/
    ├── x86_64.rtg
    ├── x86_32.rtg
    ├── aarch64.rtg
    └── arm.rtg
```

Each slice keeps an architecture beside its ABI, runtime, format, and target
compositions. The Wasm/VM family remains in
[`wasm32.rtg`](wasm32.rtg).

Generated Go is checked in so an ordinary `go build` needs no generator:

```text
go generate ./backend/definitions
go generate ./internal/targetinfo
go generate ./internal/backendcompiled
```

The first command emits each production architecture projection. The second
resolves target descriptors and writes the registry and the non-enforcing
source-volume report. The third refreshes the ordinary-Go compiled backend and
the source bundle used to prepare external definitions.

## Adding a native architecture

Add a new architecture slice under `native/` and import it from `native.rtg`;
do not add an architecture switch to the RTG parser or generator.

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
10. Add ABI, runtime, format, and target compositions.
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

`TestNativeDefinitionEmbeddedGoBudget` enforces the complete imported native
catalog at no more than 60,000 semantic Go bytes and 200 Go declarations. This
is an architecture-maintainability guard, not permission to move target
algorithms into an unmeasured file. Reusable generator code may implement a
generic format constructor; machine and OS differences must remain visible as
typed `.rtg` facts.

## Identity and pruning

The catalog hash identifies the complete expanded declaration graph and
appears in generated provenance headers. Import directories, comments, and
formatting are not semantic. An imported file's basename is semantic because
it supplies the virtual package used to resolve private helper names. A target
semantic identity hashes only the selected target and its transitive machine,
ABI, runtime, format, and embedded Go dependencies. RTGU bindings, target
descriptors, prepared artifacts, and cache keys use the target identity.

Comments, formatting, declaration order, and unreachable architectures do not
change a target identity. Changing a reachable hook or declaration does.
Fixed and prepared generation starts at one selected target and emits only
reachable Go declarations.

## Imports

`@import "relative/path.rtg"` includes another definition fragment at the top
level. Paths are resolved relative to the importing file, nested imports are
supported, and cycles are rejected. Imported files are fragments: the root
owns `definition`, `unit`, and `implements`, while the parser validates the
expanded graph as one closed catalog. Every file forms a virtual package named
after its basename: Go helpers and bounded sequences are short and local by
default, and another file can refer to one explicitly as
`package_name.helper`. Machine declarations and explicit export names remain
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
review aids. The closed native-catalog Go ceiling is separately enforced by
the RTG tests. Compiler and output performance acceptance remains exclusively
defined by the existing hard gates in `backend/main_test.go`.
