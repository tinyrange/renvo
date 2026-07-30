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

The native catalog contains shared helpers once. Architecture instruction
encoders and genuinely irregular ABI, runtime, relocation, and format
algorithms remain typed embedded Go hooks. The compiler does not analyze hook
complexity or termination.

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
- ABI inheritance, register/stack policy, save sets, and typed ABI hooks;
- runtime operation declarations and typed runtime hooks;
- output-format parameters and relocation hooks;
- one target composition across machine, ABI, runtime, and format.

The language has no macros, general evaluator, recipe interpreter, or Go
complexity analyzer.

## Embedded Go boundary

Embedded Go is parsed and checked with Renvo's Go frontend. Imports, package
clauses, compiler directives, undefined symbols, and incompatible hook
signatures are rejected.

Generation starts from one target or architecture and emits only transitively
reachable declarations. An opaque hook is appropriate when an encoder or
platform algorithm has real control flow that would be less clear as a table.
Declarative forms remain authoritative for operation identity, operands,
effects, constraints, composition, and hook placement.

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
architecture projections. The four native encoder bodies remain independently
prunable even though their source authority is one catalog.

Built-ins expose stable compiler-facing names so fixed-target compilers retain
their direct fast calls. External definitions cannot have names known to a
release compiler, so preparation emits a target-neutral adapter around the same
typed operation bindings. That adapter is a packaging boundary only: machine,
ABI, runtime, relocation, and format semantics remain in the selected
definition.

`go generate ./internal/targetinfo` resolves every public target and emits:

- frontend and backend target registries;
- target help;
- `machine-definitions.generated.md`, including non-enforcing declarative and
  reachable/catalog Go-volume metrics.

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
acceptance remains exclusively defined by `backend/main_test.go`; source-volume
metrics never loosen a gate.

## Acceptance record

The final 2026-07-30 migration run recorded:

- native authored source changed from four files totaling 255,840 bytes and
  8,408 lines to one 246,064-byte, 8,092-line catalog;
- checked-in native architecture projections changed from 148,330 bytes of
  algorithm plus test-only contract output to 43,026 bytes of production
  output, a reduction of 105,304 bytes;
- the ordinary-Go compiled backend changed by -22 bytes and its compressed
  preparation bundle by +723 bytes;
- two complete generation passes were byte-identical;
- the full backend suite passed in 10.283 seconds and the full frontend suite,
  including self-hosting, passed in 200.097 seconds under a 4 GiB memory cap;
- frontend self-host performance passed at 350 ms measured CPU, 1961/1000
  normalized calibration units, 23,140 KiB peak RSS, and a 1,430,569-byte
  stage3 compiler;
- VM32 backend self-hosting passed with a 1,131,698-byte artifact, 273,766
  execution steps, and 73,815,312 bytes peak memory;
- full frontend compilation inside VM32 passed with a 3,011,055-byte artifact,
  1,634,978-byte Linux output, 7,542,001,205 execution steps, and 140,001,892
  bytes peak memory; and
- ordinary Go builds passed for Linux amd64/arm64, Windows amd64/arm64, and
  Darwin arm64.

All fixed-target compiler runtime, RSS, and binary-size checks passed their
unchanged limits. `backend/main_test.go` was not modified.
