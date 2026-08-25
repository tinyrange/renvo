# Renvo machine definitions: implemented system

This document describes the implemented machine-definition system. The
backend-author workflow is in
[`../definitions/README.md`](../definitions/README.md).

## Current source authority

- Every `backend/definitions/<os>_<arch>.rtg` native file is an independent
  `native_v1` entrypoint for exactly one target. It imports reusable amd64,
  386, AArch64, ARM, and format fragments from the same directory.
- `backend/definitions/wasm32.rtg` is the separate `structured32` family for
  WASI, browser Wasm, and VM32.
- Generated production Go is checked in and compiled into ordinary Go-module
  builds. No generator or repository checkout is needed at runtime.
- External definitions use the same parser, resolver, validator, typed hook
  checker, reachability analysis, and direct-code generation primitives as the
  built-in catalog.

The native source set contains shared helpers once. Repeated instruction rows,
straight-line encoder/ABI/runtime sequences, operation dispatch, entry and I/O
templates, architecture label relocations, ELF/PE executable relocation
patching, ELF/PE image construction, and Linux-module ELF construction are
bounded declarations. Genuinely irregular encoders, branch relaxation, ABI
edges, and Mach-O algorithms remain typed embedded Go hooks. The compiler does
not analyze hook complexity or termination.

Top-level `@import "relative/path.rtg"` composes one target definition before
parsing and resolution. Imports are relative to the importing file, recursive,
cycle-checked, included in the same source-size and semantic Go budgets, and
source-mapped so diagnostics identify the owning fragment. Each file is a
virtual package named after its basename. Private Go helpers and bounded
sequences resolve locally and receive deterministic internal qualification;
an `extend arch` block inherits the imported ISA's symbols so target-specific
sequences can use short helper names without manual architecture prefixes.
Explicit `package_name.helper` references remain available. Machine
declarations and explicit exports remain global. Import directories and
spelling are not semantic, while the package basename is.

## Backend families

Every target descriptor names exactly one family:

- `native_v1` is a register-machine contract with explicit native instruction
  emission, relocations, ABI roles, runtime operations, and native images.
- `structured32` retains Wasm/VM lowering and module construction without
  forcing structured-control concepts into the native schema.

Family mismatches are rejected during resolution, before source generation.

## Native schema

The bounded native model includes:

- word, data-pointer, code-pointer, and function-pointer widths;
- byte order and alignment;
- physical registers;
- overlapping `register_class` subsets;
- multi-register `register_group` values;
- named `address_space` declarations, including non-power-of-two address
  widths and code/data distinctions;
- role mappings, conditions, forms, instruction rows, and typed operation
  bindings;
- bounded straight-line sequences with calls, local values, assignments, and a
  final return, but no general branches or loops;
- ABI inheritance, register/stack policy, save sets, and typed ABI hooks;
- runtime operation declarations, operation-to-sequence dispatch, entry byte
  templates, named BSS records, and typed runtime hooks;
- named label-relocation encodings for relative-32, AArch64, and ARM32 code;
- ELF/PE executable constructors, executable relocation encodings, Linux-module
  ELF construction, output-format parameters, and narrow typed format helpers;
- one target composition across machine, ABI, runtime, and format.

The language has no macros, general evaluator, recipe interpreter, or Go
complexity analyzer.

## Embedded Go boundary

Embedded Go is parsed and checked with Renvo's Go frontend. Imports, package
clauses, compiler directives, undefined symbols, and incompatible hook
signatures are rejected.

Generation starts from one target or architecture and emits only transitively
reachable declarations. An opaque hook is appropriate when an encoder or
platform algorithm has target-specific control flow that would be less clear
as a table or bounded sequence. Declarative forms remain authoritative for
operation identity, operands, effects, constraints, composition, relocation
selection, format selection, and hook placement.

`TestNativeDefinitionEmbeddedGoMetrics` reports deduplicated semantic Go bytes
and declarations across all entrypoints as a code-hygiene signal. These
figures are reviewed rather than enforced as a numeric acceptance gate. The
test separately prevents the replaced
opaque module-image, Windows/386 I/O, label-relocation, ELF-relocation, and x86
PE-relocation functions from being reintroduced. Windows/AMD64 still has a
legacy declarative `io_template` byte sequence; replacing it with named bounded
sequences is remaining maintainability work.

## Identity

There are two identities:

1. The entrypoint identity hashes its complete canonical import graph and
   records authored provenance in generated manifests.
2. The target semantic identity hashes the selected target, its reachable
   machine/ABI/runtime/format declarations, reachable bounded-sequence ASTs,
   reachable Go declarations, and compatibility versions.

RTGU bindings, public target descriptors, RTGB artifacts, and prepared cache
keys use target identity. Comments, formatting, declaration order, and an
unreachable architecture do not invalidate existing targets. Reachable
semantic changes do.

## Generated topology

`go generate ./backend/definitions` emits the checked-in production
architecture projections from four closed `*_algorithms.rtg` roots. Each root
imports only its shared ISA fragment; no operating-system entrypoint is chosen
as an accidental authority for an architecture. The four native encoder bodies
and their declarative label-relocation finalizers remain independently
prunable, while each target can evolve without affecting unrelated
operating-system code.

Built-ins expose stable compiler-facing names so fixed-target compilers retain
their direct fast calls. Shared projections replace ISA encoder algorithms.
Linux/amd64 is the first authoritative production target projection: its
runtime numbers, syscall ABI shuffles, process-stack entry template, and ELF
image integration are generated from `linux_amd64.rtg` and its imported format
definition into `compiler_linux_amd64_impl.go`. Its private symbols use a
separate built-in namespace so prepared definitions can coexist in the same
self-hosted compiler source bundle. The remaining built-in OS/runtime paths
are still handwritten pending their own equivalence migrations. External
definitions use the target-neutral prepared adapter and take runtime,
relocation, and format semantics from the selected definition.

`go generate ./internal/targetinfo` resolves every public target and emits:

- frontend and backend target registries;
- target help;
- `machine-definitions.generated.md`, including per-target review metrics for
  declarative and reachable/entrypoint Go volume.

`go generate ./internal/backendcompiled` emits:

- the ordinary-Go compiled backend package;
- a compressed compiler-source bundle used for external backend preparation.

The compiler source declares its zero fixed-target value directly. The bundle
generator parses the single `renvoPreparedBackend` constant in the target
policy and specializes its value to one in the checked-in preparation bundle.
It rejects a missing, ambiguous, or structurally invalid setting. Runtime
preparation performs no source replacement; the ordinary compiled backend
retains mode zero.

## Custom preparation

For an external `.rtg` definition Renvo:

1. parses, resolves, and validates the selected target;
2. emits one reachable target-specialized backend;
3. combines it with the checked-in preparation source bundle;
4. self-compiles that backend with the built-in host compiler;
5. stores a host-specific linked image in an RTGB artifact;
6. caches it by semantic and compatibility identity;
7. executes it through the prepared-backend protocol.

No host Go toolchain is invoked. A target unknown to the release registry can
be prepared and can emit Linux ELF, Windows PE, Darwin Mach-O, and the
supported 32-bit native outputs.

## Stress fixtures

`internal/rtg/testdata/native_8086.rtg` and `native_avr.rtg` are schema design
fixtures, not product backends. They resolve and generate direct Go for
representative move, load/store, arithmetic, branch, call, and relocation
operations while exercising:

- 8- and 16-bit words;
- segmented 20-bit addressing;
- Harvard code/data address spaces;
- overlapping constrained register sets;
- register pairs;
- short control transfers and restricted encodings.

## Verification

The generated production encoders, target registries, bundled compiler, custom
preparation, RTGU/RTGB binding, Wasm/VM family, native target suites, and full
self-hosting suite are exercised by the repository tests. Performance
acceptance remains exclusively defined by `backend/main_test.go`; native
embedded-Go volume is review evidence and never loosens a compiler or output
gate.

## Acceptance record

The native source set is checked as the deduplicated union of every independent
target entrypoint. Its current semantic Go bytes and declarations are reported
by `TestNativeDefinitionEmbeddedGoMetrics` without a numeric pass/fail ceiling.
The generated target report separately shows the reachable and
complete-entrypoint volume retained by each target.

Prepared production checks cover AArch64, Windows PE on amd64/386/arm64, Linux
kernel-module ELF, and Linux 386/ARM outputs. Generation, complete backend,
complete frontend/self-hosting, cross-build, and deterministic-output results
are recorded from the final validation run for each change rather than kept as
stale historical numbers here. `backend/main_test.go` remains unmodified.
