# Backend definitions

Renvo has two backend families:

- `native_v1` describes register machines that emit native code, relocations,
  ABI calls, hosted runtime operations, and native images.
- `structured32` contains the existing Wasm and VM32 implementation. It is
  deliberately separate because structured control flow, module index spaces,
  and validation are not native-machine concepts.

The built-in native source of truth is [`native.rtg`](native.rtg). All nine
built-in native targets are compositions in that one closed catalog. The
Wasm/VM family remains in [`wasm32.rtg`](wasm32.rtg).

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

Add the machine to `native.rtg`; do not add an architecture switch to the RTG
parser or generator.

1. Declare the physical registers once with `registers`.
2. Use `register_class` for overlapping constrained subsets.
3. Use `register_group` for multi-register values such as register pairs.
4. Declare non-flat storage with one or more `address_space` blocks.
5. Map compiler roles in `locations`; names such as `primary` and `stack` are
   roles, not required physical register spellings.
6. Describe recurring encodings in `forms` and instruction variants as rows in
   `instructions`.
7. Bind every `direct_emitter_v1` operation to an instruction or typed Go hook,
   or explicitly reject an operation supported by a later contract.
8. Add ABI, runtime, format, and target compositions.
9. Export only the architecture algorithms used by the checked-in compiler.
10. Add the architecture projection to `generate.go`, regenerate, and run the
    complete backend and frontend suites under the repository memory cap.

Embedded Go is an opaque, typed escape hatch. It is appropriate for irregular
encoders, immediate synthesis, relocation patching, ABI edges, and runtime or
format algorithms with real control flow. Renvo checks hook signatures but
does not attempt complexity or termination analysis. Before adding a hook,
check whether a table row, an existing form, or a shared catalog helper states
the difference more directly.

## Identity and pruning

The catalog hash identifies the complete authored `.rtg` file and appears in
generated provenance headers. A target semantic identity hashes only the
selected target and its transitive machine, ABI, runtime, format, and embedded
Go dependencies. RTGU bindings, target descriptors, prepared artifacts, and
cache keys use the target identity.

Comments, formatting, declaration order, and unreachable architectures do not
change a target identity. Changing a reachable hook or declaration does.
Fixed and prepared generation starts at one selected target and emits only
reachable Go declarations.

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

The metrics in `backend/docs/machine-definitions.generated.md` are review aids.
Only the existing hard gates in `backend/main_test.go` decide acceptance.
