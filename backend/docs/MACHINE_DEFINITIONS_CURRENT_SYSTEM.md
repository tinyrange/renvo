# Renvo machine definitions: implemented system

This document supersedes the pre-refactor meeting brief. The design and
migration rationale are in [`PLAN.html`](PLAN.html); the backend-author
workflow is in [`../definitions/README.md`](../definitions/README.md).

## Current source authority

- `backend/definitions/native.rtg` is the single closed built-in `native_v1`
  catalog for amd64, 386, AArch64, and ARM across Linux, Windows, Darwin, and
  Linux kernel-module compositions.
- `backend/definitions/wasm32.rtg` is the separate `structured32` family for
  WASI, browser Wasm, and VM32.
- Generated production Go is checked in and compiled into ordinary Go-module
  builds. No generator or repository checkout is needed at runtime.
- External definitions use the same parser, resolver, validator, typed hook
  checker, reachability analysis, and direct-code generation primitives as the
  built-in catalog.

The native catalog contains shared helpers once. Repeated instruction rows,
straight-line encoder/ABI/runtime sequences, operation dispatch, entry and I/O
templates, architecture label relocations, ELF/PE executable relocation
patching, ELF/PE image construction, and Linux-module ELF construction are
bounded declarations. Genuinely irregular encoders, branch relaxation, ABI
edges, and Mach-O algorithms remain typed embedded Go hooks. The compiler does
not analyze hook complexity or termination.

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

The closed production native catalog is hard-limited by
`TestNativeDefinitionEmbeddedGoBudget` to 60,000 semantic Go bytes and 200 Go
declarations. The test is deliberately a simple size/count guard, not a Go
complexity analyzer. It also prevents the replaced opaque module-image,
Windows-I/O, label-relocation, ELF-relocation, and x86 PE-relocation functions
from being reintroduced.

## Identity

There are two identities:

1. The catalog identity hashes the complete canonical source file and records
   authored provenance in generated manifests.
2. The target semantic identity hashes the selected target, its reachable
   machine/ABI/runtime/format declarations, reachable Go declarations, and
   compatibility versions.

RTGU bindings, public target descriptors, RTGB artifacts, and prepared cache
keys use target identity. Comments, formatting, declaration order, and an
unreachable architecture do not invalidate existing targets. Reachable
semantic changes do.

## Generated topology

`go generate ./backend/definitions` emits the checked-in production
architecture projections. The four native encoder bodies and their declarative
label-relocation finalizers remain independently prunable even though their
source authority is one catalog.

Built-ins expose stable compiler-facing names so fixed-target compilers retain
their direct fast calls. External definitions cannot have names known to a
release compiler, so preparation emits a target-neutral adapter around the same
typed operation bindings. That adapter is a packaging boundary only: machine,
ABI, runtime, relocation, and format semantics remain in the selected
definition.

`go generate ./internal/targetinfo` resolves every public target and emits:

- frontend and backend target registries;
- target help;
- `machine-definitions.generated.md`, including per-target review metrics for
  declarative and reachable/catalog Go volume.

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
acceptance remains exclusively defined by `backend/main_test.go`; the native
catalog ceiling adds a maintainability constraint and never loosens a compiler
or output gate.

## Acceptance record

The current closed native catalog is 217,610 authored bytes over 7,255 lines.
The generated catalog report measures 57,334 semantic Go bytes and 139
declarations; the stricter declaration-source audit measures 59,374 bytes and
139 declarations. Both are below the enforced 60,000/200 ceiling. The four
checked-in native algorithm projections total 44,824 bytes.

Prepared production checks cover AArch64, Windows PE on amd64/386/arm64, Linux
kernel-module ELF, and Linux 386/ARM outputs. Generation, complete backend,
complete frontend/self-hosting, cross-build, and deterministic-output results
are recorded from the final validation run for each change rather than kept as
stale historical numbers here. `backend/main_test.go` remains unmodified.
