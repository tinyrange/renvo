# Renvo machine definitions: current system and meeting brief

Status: current implementation on `agent/machine-definitions`, inspected after
commit `5df4ea26` on 30 July 2026.

This document describes the machine-definition system as it exists now. It is
not a restatement of the intended design in `PLAN.html`. Its purpose is to let a
group enter an architecture meeting with a common model of:

- what is implemented;
- what the source of truth is;
- how built-in and custom backends are compiled and executed;
- why the implementation contains so much Go;
- which Go is authored, generated, duplicated, or stored as data;
- where the current implementation differs from the intended end state;
- what can be simplified without changing the product;
- which simplifications require changing an earlier design decision.

The short version is:

> The implementation successfully moved backend ownership into closed `.rtg`
> files, but it did not move backend algorithms away from Go. About 83% of the
> built-in definition text is embedded Go. The repository then projects the
> same backend source into several additional Go representations so ordinary Go
> builds, self-hosted builds, and custom-backend preparation can all use it.
> There is also still a dual production topology: built-in targets use the
> architecture-specific compiler paths, while prepared custom targets use the
> new generic RTG adapter.

That distinction is important. "Too much Go" can refer to at least three
different problems:

1. too much Go for a human backend author to write;
2. too much generated Go checked into the repository;
3. too much duplicated Go machinery in the compiled architecture.

Those problems have different solutions. Removing generated files does not make
the design less dependent on Go. Adding a richer `.rtg` DSL can reduce authored
Go, but can also recreate an instruction interpreter or a second programming
language. Refactoring the compiler into an importable package can remove the
largest generated copy without changing `.rtg` at all.

## 1. The product model

Renvo has two backend delivery modes.

### 1.1 Built-in backends

The release compiler contains all advertised backends. A normal Go module build
must work without running a generator and without requiring a separately
installed backend. Generated source is therefore checked in.

The current built-in target set is:

| Target | Definition | Architecture family | Output |
|---|---|---|---|
| `linux/amd64` | `amd64.rtg` | x86-64 | ELF executable/object |
| `windows/amd64` | `amd64.rtg` | x86-64 | PE executable/COFF object |
| `linux-kernel/amd64` | `amd64.rtg` | x86-64 | relocatable ELF kernel module |
| `linux/386` | `386.rtg` | x86-32 | ELF executable |
| `windows/386` | `386.rtg` | x86-32 | PE executable |
| `linux/aarch64` | `aarch64.rtg` | AArch64 | ELF executable |
| `darwin/arm64` | `aarch64.rtg` | AArch64 | Mach-O executable |
| `windows/arm64` | `aarch64.rtg` | AArch64 | PE executable |
| `linux/arm` | `arm.rtg` | ARM32 | ELF executable |
| `wasi/wasm32` | `wasm32.rtg` | VM32 lowering | Wasm module |
| `browser/wasm32` | `wasm32.rtg` | VM32 lowering | packaged HTML/Wasm |
| `vm/vm32` | `wasm32.rtg` | VM32 lowering | RNVM bytecode |

The definition hash covers the complete closed `.rtg` file, including embedded
Go. Multiple targets that share one definition therefore share the definition
hash.

### 1.2 Custom backends

A user may supply a closed definition:

```text
renvo -backend machines.rtg -t vendor/new-target ...
```

or explicitly prepare it:

```text
renvo backend build machines.rtg \
  -t vendor/new-target \
  -o vendor-new-target.rtgb
```

The custom target does not need to be known by the release compiler. Renvo:

1. parses and validates the definition;
2. generates a target-specialized Go backend;
3. compiles that backend with Renvo itself for the current host;
4. places the host-native linked image in an `.rtgb` artifact;
5. caches the artifact by definition and compiler compatibility;
6. runs the prepared backend to compile bound frontend units.

The host Go toolchain is not invoked during this workflow.

## 2. Vocabulary

The implementation uses several similar-sounding artifacts.

### `.rtg`

A closed, versioned machine-definition source file. It contains declarative
architecture, ABI, runtime, format, and target declarations plus one embedded
`go backend` block. A definition does not import another `.rtg` file.

### RTGU

The canonical linked frontend unit consumed by a backend. It contains a target
binding:

- canonical target name;
- 32-byte definition hash;
- descriptor version.

The binding prevents a unit type-checked for one target definition from being
silently compiled by another.

### Generated backend source

Ordinary Go source emitted from a resolved definition. It contains constants,
typed direct-emitter wrappers, selected embedded Go declarations, ABI/runtime/
format adapters, and target-specialized facts.

### Architecture algorithm projection

The checked-in Go file containing the embedded algorithms exported by a
built-in definition under the stable names expected by the existing compiler.
Examples are:

- `backend/compiler_aarch64_impl.go`;
- `backend/compiler_amd64_target_impl.go`;
- `backend/compiler_386_target_impl.go`;
- `backend/compiler_arm_impl.go`;
- `backend/compiler_wasm32_impl.go`.

### Architecture contract projection

The checked-in `backend/rtg_*_contract_generated.go` files. These contain the
typed register facts, operation bindings, and direct wrappers for
`direct_emitter_v1`.

At present these files compile in host Go backend tests but are not listed in
`backend/compiler_sources.txt`, so they do not drive the built-in compiler
bundle. This is a central fact about the current topology.

### `.rtgb`

A prepared, single-target backend artifact. It contains a public target
descriptor, compatibility versions, host identity, and a native linked-image
payload.

### Compiled backend bundle

`internal/backendcompiled` is the normal-Go package used by `cmd/renvo`. It
contains a generated copy of the backend compiler as ordinary package code and
a second compressed copy of its source for custom-backend preparation.

## 3. The `.rtg` language as implemented

Every machine definition starts with:

```text
definition 1
unit <name>
implements direct_emitter_v1
```

The parser recognizes these top-level declaration kinds:

- `system`;
- `ir`;
- `arch`;
- `abi`;
- `runtime`;
- `format`;
- `target`;
- `go backend`.

The built-in definitions primarily use `arch`, `abi`, `runtime`, `format`,
`target`, and `go backend`.

### 3.1 Declarative responsibility

The declarative portion describes facts and composition:

- registers and special register roles;
- conditions;
- instruction forms and instruction rows;
- bindings from stable IR operations to instructions or Go algorithms;
- ABI register and stack policy;
- runtime operation support and hooks;
- image and object format hooks;
- relocation policy;
- pointer widths, integer widths, alignment, and endianness;
- target names, aliases, build tags, capabilities, output kinds, and default
  arena sizes.

Resolution turns named references into a typed `TargetDescriptor`. Validation
checks that architecture, ABI, runtime, and format choices agree.

### 3.2 Algorithmic responsibility

The `go backend` block contains ordinary package-level Go declarations written
in Renvo's supported Go subset. In the current definitions it owns:

- instruction encoding;
- register and address lowering;
- immediate synthesis;
- branch and relocation patching;
- call and frame mechanics;
- ABI shuffles;
- syscall and hosted-runtime entry/exit;
- ELF, PE, COFF, Mach-O, Wasm, RNVM, and kernel-module image construction;
- Windows import and command-line handling;
- architecture-specific optimizations.

Embedded Go is not treated as an unparsed string fragment. The RTG parser
extracts the balanced block, sends it through the existing Renvo Go parser and
checker with a synthetic RTG API prelude, records symbols, and emits reachable
declarations with deterministic mangling.

### 3.3 The RTG API

Embedded algorithms compile against a generated API centered on:

- `RTGEmitter`;
- `RTGRegister`;
- `RTGLabel`;
- `RTGAddress`;
- `RTGCondition`;
- byte and integer append/patch helpers;
- code, data, BSS, label, relocation, and symbol operations;
- static, dynamic, and external import operations;
- kernel metadata access;
- bounded scratch and alignment helpers.

The type-checking prelude is in `internal/rtg/typecheck.go`. The executable
implementation of that API is emitted into
`backend/compiler_rtg_generated_impl.go` for host Go and into a prepared
backend's generated source.

### 3.4 The direct-emitter contract

`direct_emitter_v1` is currently intrinsic to the generator. Its schema and
effect information live in `internal/rtg/contract.go`, not in each machine
definition.

It defines 38 operations, including:

- register moves and immediate moves;
- address formation;
- signed and unsigned loads;
- stores;
- integer arithmetic and bit operations;
- comparisons and condition materialization;
- calls, indirect calls, jumps, conditional jumps, return, and leave;
- host syscall emission;
- variable shifts and signed division;
- byte copying.

Each architecture must either bind every operation or explicitly reject an
operation that the contract permits to be rejected. Validation checks:

- completeness;
- duplicates;
- referenced instruction or Go algorithm existence;
- Go signature compatibility;
- effect-contract completeness.

The effect table is generation-time metadata. A fixed generated backend gets
direct calls rather than a runtime operation table or recipe interpreter.

### 3.5 Parsing and validation boundaries

The definition core deliberately has no filesystem, process, cache, or compiler
policy. Its public flow is byte-slice based:

```text
Parse(source, filename)
  -> ResolveDefinitions(document)
  -> GeneratePreparedBackend(resolved, target)
```

Current resource limits include:

- 16 MiB source;
- 262,144 tokens;
- 4,096 declarations;
- 16,384 tokens in one field value;
- 256 levels of statement nesting;
- 8,192 statements in one body.

Embedded package clauses and imports are rejected. Type checking uses the Renvo
frontend rather than `go/types`. The definition hash is canonicalized so
formatting and comments do not change identity, while executable changes do.

## 4. Built-in generation workflow

The generation entry points are recorded in
`backend/definitions/generate.go`.

For each architecture, the workflow reads its closed `.rtg` file and emits two
different projections:

```text
                          +------------------------------+
                          | backend/definitions/*.rtg    |
                          | declarations + embedded Go   |
                          +---------------+--------------+
                                          |
                      +-------------------+-------------------+
                      |                                       |
                      v                                       v
       architecture algorithm projection        architecture contract projection
       compiler_<arch>_impl.go or                rtg_<arch>_contract_generated.go
       compiler_<arch>_target_impl.go
```

The generator also emits:

- `compiler_rtg_generated_impl.go`: the full RTG emitter kernel for host Go;
- `compiler_rtg_inactive_impl.go`: Renvo-build stubs used when prepared-backend
  branches must type-check but cannot execute.

`go generate ./internal/targetinfo` separately:

1. parses all built-in definitions;
2. resolves their target descriptors;
3. combines them with target release policy;
4. generates `internal/targetinfo/registry_generated.go`;
5. generates frontend target help;
6. generates `backend/docs/machine-definitions.generated.md`.

`go generate ./internal/backendcompiled` then reads
`backend/compiler_sources.txt` and emits:

- `internal/backendcompiled/compiler_generated.go`;
- `internal/backendcompiled/sources_generated.go`.

This last generation step is the source of most of the visible Go expansion.

## 5. Normal built-in compilation flow

The normal command starts with `backendcompiled.Backend`:

```text
Go source
  |
  v
driver resolves target and source tags
  |
  v
frontend parses, checks, links, and serializes RTGU
  |
  v
frontend appends built-in target name + definition hash + descriptor version
  |
  v
backendcompiled.Backend verifies RTGU binding
  |
  v
compiled-in backend parses RTGU and selects architecture-specific compiler
  |
  v
native executable, object, Wasm, HTML/Wasm, or RNVM output
```

The frontend registry is descriptor-first. Build tags, integer and pointer
widths, target OS, capabilities, output kind, and arena defaults come from the
resolved/generated target data rather than guessing from the target spelling.

The backend also verifies the binding itself. A frontend unit cannot merely say
`linux/amd64`; it must carry the definition identity expected by that target in
the compiled backend.

### 5.1 What actually dispatches built-ins

Built-in production dispatch currently remains architecture-specific:

```text
renvoTryCompileScalarProgramAmd64Cached
renvoTryCompileScalarProgram386Cached
renvoTryCompileScalarProgramAarch64Cached
renvoTryCompileScalarProgramArmCached
renvoTryCompileScalarProgramWasm32
```

The exported encoder algorithms called by these paths are now generated from
the `.rtg` embedded Go blocks. That means definition ownership is real: editing
the algorithm in a built-in definition and regenerating changes production
encoding.

However, the complete `rtg_<arch>_contract_generated.go` projection is not in
`compiler_sources.txt`. Therefore built-ins do not currently enter through the
same generic `renvoTryCompileScalarProgramRTG` adapter used by prepared custom
backends.

This is not merely a naming detail. It means the system has:

- one architecture-specific dispatch topology for built-ins;
- one generated direct-emitter topology for custom prepared backends;
- conditionals in the shared compiler to select the prepared topology.

Across backend implementation files there are currently 140 references to
`renvoPreparedBackend`, 128 of them in `compiler_common_impl.go`.

## 6. Custom definition resolution and frontend binding

When `-backend` is present, target resolution happens before source selection.
That is necessary because an unknown custom target may define:

- its OS;
- source filename selection;
- build tags;
- integer and pointer widths;
- maximum alignment;
- runtime capabilities;
- output kind;
- default arena size.

The sequence is:

1. `internal/driver/backend_definition_host.go` locates `-backend`, `-system`,
   and `-t`.
2. The selected `.rtg` or `.rtgb` file is read.
3. `internal/backenddef.Resolve` obtains a public descriptor.
4. The descriptor supplies the frontend target facts and build tags.
5. The frontend compiles the application.
6. The resulting RTGU binding is replaced with the custom target name,
   definition hash, and descriptor version.
7. The backend later verifies the same binding before code generation.

An explicit `-t` remains authoritative. A system profile supplies a target only
when `-t` is absent.

## 7. Custom backend preparation

`internal/backendjit.Prepare` performs preparation.

```text
closed machines.rtg
  |
  v
parse -> resolve -> validate
  |
  v
GeneratePreparedBackend(selected target)
  |
  v
one target-specialized Go source file
  |
  +---------------- compressed checked-in backend compiler sources
  |
  v
driver.CompileUnit(..., bootstrap = backendcompiled.Backend)
  |
  v
host-native Renvo linked image
  |
  v
RTGB metadata + payload
  |
  v
content-addressed cache / explicit .rtgb file
```

### 7.1 Source specialization

Preparation reconstructs the 33 backend compiler sources embedded in
`internal/backendcompiled/sources_generated.go`.

It excludes the normal full and inactive RTG kernels, then appends the generated
single-target implementation. Two source edits specialize the compiler:

- `var renvoFixedTarget int` becomes `var renvoFixedTarget int = 0`;
- `const renvoPreparedBackend = 0` becomes
  `const renvoPreparedBackend = 1`.

These are currently exact textual replacements guarded by occurrence counts.
They are deterministic, but they are also a coupling between the preparation
layer and declarations in backend source.

### 7.2 Bootstrap

The reconstructed compiler source and generated target source are passed to the
normal frontend driver. `backendcompiled.Backend` compiles that compiler for the
current host with:

- stripping enabled;
- linked-image output enabled;
- the current host target fixed;
- no host Go compiler.

The produced linked image is the executable payload of the prepared backend.

### 7.3 Cache identity

The cache key contains:

- definition identity;
- exact target name;
- exact host name;
- generator version;
- backend kernel version;
- RTGU version;
- prepared protocol version;
- optimization version.

Filesystem publication writes and syncs a temporary file, closes it, and
renames it into place. An `ArtifactCache` interface lets embedders provide a
different store.

## 8. Prepared artifact and execution protocol

The `.rtgb` codec is in `internal/rtgb`.

The version-2 artifact has:

- `RTGB` magic;
- fixed-size header;
- bounded metadata and payload lengths;
- definition hash;
- target descriptor;
- host tuple;
- generator, kernel, protocol, RTGU, and optimization versions;
- Adler-style payload checksum;
- a Renvo linked-image payload.

Current bounds are:

- 1 MiB metadata;
- 64 MiB payload.

The default process runner:

1. validates all protocol versions;
2. decodes the linked image;
3. creates a temporary protocol directory;
4. writes the bound RTGU to `input.rtgu`;
5. executes the host-native prepared image;
6. passes target and output options on its argument vector;
7. reads `output.bin`;
8. removes the temporary directory.

The `Runner` interface separates preparation from execution so an embedder can
replace process execution. The default protocol is still filesystem-mediated
and process-shaped even when the linked-image runtime can execute native code
in-process on a supported host.

## 9. Where the Go is

There are four materially different kinds of Go in this implementation.

### 9.1 Authored infrastructure Go

This is the actual implementation of the RTG language and workflow:

- scanner and parser;
- resolver and validator;
- canonicalizer and hasher;
- embedded-Go symbol and type analysis;
- source generator;
- target registry generator;
- RTGU binding;
- RTGB codec;
- custom preparation, cache, and runner;
- frontend integration.

The production files in `internal/rtg` are about 7,200 lines before tests. The
package is large because it implements both a language frontend and a source
emitter without using host-only Go parsing, type-checking, or formatting APIs
in its portable core.

### 9.2 Embedded authored Go

The five built-in `.rtg` files contain 11,657 lines total:

| Definition | Total lines | `go backend` block | Declarative/header | Go-block share |
|---|---:|---:|---:|---:|
| `386.rtg` | 1,489 | 1,273 | 216 | 85.5% |
| `aarch64.rtg` | 3,073 | 2,752 | 321 | 89.6% |
| `amd64.rtg` | 2,463 | 1,820 | 643 | 73.9% |
| `arm.rtg` | 1,383 | 1,133 | 250 | 81.9% |
| `wasm32.rtg` | 3,249 | 2,747 | 502 | 84.5% |
| **Total** | **11,657** | **9,725** | **1,932** | **83.4%** |

The Go-block count includes the `go backend` wrapper. It is still an accurate
measure of the authoring balance.

There are 803 embedded functions across the five definitions. Normalizing the
architecture prefixes reveals repeated roles such as move, load, store, frame,
call, division, shifting, relocation patching, ELF construction, PE
construction, runtime operations, and entry handling. Some repetition is
genuinely architecture-specific. Some is schema-shaped boilerplate that could
be generated from smaller declarations.

### 9.3 Generated executable Go

The checked-in generated projections currently include:

| Projection | Lines | Bytes | Purpose |
|---|---:|---:|---|
| `compiler_generated.go` | 38,473 | 1,109,578 | normal-Go compiled backend package |
| `sources_generated.go` | 294 | 413,638 | compressed backend sources for JIT preparation |
| five architecture contract files | 3,417 | 129,745 | typed full direct-emitter contracts |
| five production algorithm files | 3,688 | 115,685 | embedded algorithms under stable legacy names |
| full RTG kernel | 702 | 18,393 | executable emitter API/adapters |
| inactive RTG kernel | 287 | 10,497 | self-host type-check stubs |
| generated target registry | 197 | 15,613 | frontend-visible target facts |

Line count particularly exaggerates `compiler_generated.go`, but the byte count
shows that the repository is also carrying substantial duplicate text.

### 9.4 Go source stored as data

`sources_generated.go` is Go syntax around compressed/base64 text. Its 294
physical lines represent roughly 414 KiB because each generated constant line
is long.

This source bundle exists so a regular Go-built Renvo executable can JIT-compile
a custom backend without finding a repository checkout. It is not executed as
Go by the host compiler; Renvo decompresses it and compiles it later.

## 10. Why the same backend appears several times

Today one backend compiler source can exist in these representations:

1. authored source in `backend/compiler_*_impl.go`;
2. algorithm source embedded in `backend/definitions/*.rtg`;
3. generated algorithm projection back into `backend/compiler_*_impl.go`;
4. generated full contract projection in `backend/rtg_*_contract_generated.go`;
5. compacted ordinary-Go copy in
   `internal/backendcompiled/compiler_generated.go`;
6. compressed source-data copy in
   `internal/backendcompiled/sources_generated.go`;
7. target-specialized generated source during custom preparation;
8. compiled native payload inside an `.rtgb` artifact.

Not every line appears in every representation, but this is the source of the
overall multiplication.

Each representation was introduced for a concrete constraint:

- the `.rtg` file must own the architecture;
- ordinary Go builds must contain built-in backends;
- fixed Renvo builds must prune irrelevant source;
- custom definitions must compile without host Go or a checkout;
- prepared artifacts must run without retaining source;
- generated code must be reviewable and stale-checkable.

The multiplication is therefore understandable, but it is not necessarily the
smallest topology that satisfies those constraints.

## 11. What is working well

The implementation has several strong properties worth preserving.

### Closed and hashable authority

A built-in or third-party definition is one file. Its executable algorithms and
composition facts share one identity. There is no ambient architecture plugin
lookup.

### Frontend facts are explicit

An unknown target can affect file selection and type layout before the frontend
runs. Target OS is not guessed from a slash-separated name.

### Bound frontend/backend contract

The RTGU definition binding prevents a prepared backend from consuming a unit
compiled under a different descriptor.

### No recipe interpreter in the code-generation hot path

Prepared source contains direct Go calls and constants. There is no runtime
property lookup, instruction-object tape, or generic recipe VM.

### No host Go requirement for custom backends

The release compiler carries enough source and compiled bootstrap machinery to
prepare a new backend itself.

### Deterministic generation and identity

Definitions are canonicalized, embedded Go is parsed, source emission is
deterministic, and generated outputs carry definition hashes.

### Full target shape

The current definitions cover native executables, object-like output, Windows
imports, Darwin, Linux kernel modules, Wasm, browser packaging, and VM32. This
is not an architecture-only proof that ignores runtime and image construction.

## 12. Current architectural problems

### 12.1 The definitions are mostly Go containers

Moving 9,725 lines of algorithms into `go backend` blocks gives the definitions
ownership, but it does not make the backend format substantially declarative.
A new architecture author still needs to understand:

- Renvo's Go subset;
- the RTG emitter API;
- the direct-emitter contract;
- ABI and runtime conventions;
- executable formats;
- integration naming and export rules.

This may be acceptable if `.rtg` is intentionally a closed packaging format
with declarative composition around Go algorithms. It is not acceptable if the
goal is for `.rtg` to be a compact target-description language.

The meeting must decide which product is intended.

### 12.2 Built-in and custom backends use different production dispatch

The full generated contract files are not in the built-in compiler manifest.
Built-ins continue through architecture-specific compiler entry points.
Prepared targets go through `renvoTryCompileScalarProgramRTG`.

Consequences include:

- 140 prepared-mode condition references;
- parallel ways to lower the same frontend semantics;
- full generated contracts that are compiled but not authoritative for built-in
  output;
- more difficult equivalence reasoning;
- more generated source representations;
- risk that a built-in target works while an equivalent external definition
  fails, or vice versa.

This is the largest structural issue in the current implementation.

### 12.3 The compiled backend package is produced by source copying

`backend` is a `package main` compiler with a constrained file layout.
`cmd/renvo` cannot import it as a normal library. The current generator solves
that by stripping the package declaration and copying the source into
`internal/backendcompiled`.

That works, but produces a 38,473-line generated file and makes source identity
checks necessary.

### 12.4 JIT preparation carries a second textual compiler

The custom-backend path needs backend compiler source, not merely the already
compiled built-in backend. The normal executable therefore contains both:

- a compiled backend implementation;
- compressed source for another backend compilation.

This is a genuine runtime capability cost, not just repository aesthetics.

### 12.5 Preparation specializes declarations with text replacement

The replacement is narrow and checked, but it is still an implementation-level
protocol expressed as exact source spelling. A typed generation input or build
configuration would be easier to evolve safely.

### 12.6 The inactive kernel is a topology workaround

`compiler_rtg_inactive_impl.go` emits inert declarations under the `renvo`
build tag so shared source containing prepared branches can type-check in a
topology where those branches are compile-time unreachable.

This is a symptom of compiling one shared source graph for two materially
different backend modes.

### 12.7 Definition ownership and shared primitives are in tension

The plan says a definition owns encoding, ABI, runtime, relocation, and image
layout, while only target-neutral primitives are shared. That avoids a hidden
backend.

It also means ELF, PE, hosted runtime, and common direct-emitter patterns are
repeated across closed files. Moving more of those algorithms into a shared RTG
library reduces definition Go, but weakens closed-file authority unless the
shared API is versioned and treated as part of the language.

### 12.8 The intrinsic contract is already an external dependency

Definitions say `implements direct_emitter_v1`, whose schema lives in the
compiler. They are therefore closed with respect to definition files, but not
self-describing in the absolute sense. They already depend on a versioned
compiler-provided contract and RTG API.

This matters because a small standard algorithm library would not introduce the
first external dependency; it would expand an existing one. The question is
whether that expansion remains transparent and target-neutral.

## 13. How much Go can be removed

There is no single percentage because different reductions solve different
problems. The useful ranges are below.

### 13.1 Repository Go volume: large reduction is possible

If the goal is to make the pull request and repository reviewable, more than
40,000 checked-in generated Go lines are potentially removable:

- 38,473 lines from `compiler_generated.go`;
- 3,417 lines from separate architecture contract projections;
- 287 lines from the inactive kernel;
- most of the 294-line source-data wrapper.

That is approximately 89% of the 47,058 generated-projection lines listed
above.

This does **not** mean 89% less Go in the architecture. Achieving the largest
part requires an importable or linkable backend-core topology so ordinary Go
builds do not need a copied package. The compiler would still be implemented in
Go.

### 13.2 Embedded backend Go: moderate reduction is realistic

A focused schema pass could likely remove roughly 2,000 to 3,500 of the 9,725
embedded Go-block lines without creating a general-purpose DSL. Candidates are:

- repeated fixed instruction wrappers;
- width-specialized load/store variants;
- register-role adapters;
- common frame/call wrapper shapes;
- table-shaped syscall and import declarations;
- declarative executable header and section facts;
- relocation records whose behavior is simple append/patch arithmetic;
- repeated target entry/exit selection where the actual algorithm is already
  shared inside the file.

This is an estimate, not a measured prototype. It corresponds to roughly
20–35% of the embedded Go, leaving approximately 6,000–8,000 lines of genuinely
algorithmic implementation.

Trying to remove most of the remaining Go would require one of:

- a much richer macro/rewrite language;
- a backend bytecode;
- a recipe interpreter;
- large compiler-provided architecture libraries;
- another general-purpose embedded language.

Those options either reproduce Go badly, hide backend ownership, or violate the
direct-generation performance model.

### 13.3 Authored RTG infrastructure Go: limited reduction is possible

The scanner, parser, resolver, validator, Go type-check integration, and source
emitter need an implementation language. In this repository that language is
Go, and the explicit requirement was to use the existing Go parser and emit Go.

Some generator code can become schema-driven. For example:

- define operation signatures and effects once in a compact checked-in schema;
- generate validator and emitter fragments from that schema;
- use one typed source writer instead of several append-oriented emitters;
- generate descriptor codecs and compatibility fields from one model.

This might remove 1,000–2,000 authored lines and, more importantly, reduce
contradictory logic. It does not remove the dependency on Go as the
implementation language.

### 13.4 Runtime dependence on Go: already lower than it looks

A prepared backend executes native code produced by Renvo. It does not load the
Go toolchain or interpret Go at runtime.

Go is currently:

- the source language of the generator;
- the embedded algorithm language;
- the generated interchange before Renvo compilation;
- the host package language for ordinary Go module builds.

It is not the runtime execution environment of a prepared backend payload.

### 13.5 Eliminating Go emission would be a design reversal

The following requirements jointly force ordinary Go generation:

- built-ins are generated and checked in;
- regular Go module builds compile them in;
- generated code has direct calls and constants;
- no backend recipe interpreter;
- custom definitions are compiled by Renvo;
- embedded algorithms use the supported Go subset.

Changing the generated target from Go to an RTG bytecode, C, assembly, or a
custom IR is possible only by revisiting one or more of those requirements.

## 14. Simplification options

The options below are ordered from least to most disruptive.

### Option A: consolidate generated projections

Generate one authoritative production file per architecture containing:

- exported algorithms;
- register and condition facts;
- complete direct-emitter bindings;
- ABI/runtime/format hooks needed by built-in compositions.

Stop checking a separate contract-only file that is not part of the release
compiler manifest. Generate contract test fixtures into temporary directories
when a test needs to inspect them.

Benefits:

- removes about 3,400 checked-in Go lines immediately;
- makes "generated contract" mean production code;
- reduces stale-output surfaces;
- makes review easier.

Limitations:

- does not eliminate the legacy-vs-prepared dispatcher;
- does not reduce embedded Go;
- requires care to preserve fixed-target pruning.

### Option B: make built-ins use the same generated direct-emitter path

Generate built-in target compositions against the same adapter used by prepared
backends. The target selector chooses a generated composition, but frontend
semantic lowering calls one stable direct-emitter interface.

The desired shape is:

```text
shared frontend semantic lowering
               |
               v
stable direct-emitter calls
               |
       +-------+--------+
       |                |
       v                v
generated built-in   generated custom
fixed composition    fixed composition
```

This should remove most `renvoPreparedBackend` branches and make built-in
equivalence meaningful by construction.

Benefits:

- one production topology;
- generated contracts become authoritative;
- less conditional code in `compiler_common_impl.go`;
- fewer custom-only adapters;
- easier new-architecture bring-up.

Risks:

- it touches the backend hot path;
- generated built-in size and performance must be measured carefully;
- existing specialized architecture paths contain optimizations that must
  survive;
- fixed-target source pruning must remain effective.

This is the highest-value structural change.

### Option C: replace copied package source with an importable compiler kernel

Refactor the backend compiler core into a normal internal package that both:

- `backend/compiler_main.go` can wrap for standalone/self-host builds;
- `internal/backendcompiled` or `cmd/renvo` can import for ordinary Go builds.

Generated architecture files would compile in that package rather than being
textually copied into `compiler_generated.go`.

Benefits:

- removes the 38,473-line generated compiler copy;
- normal Go tooling sees real source locations;
- eliminates digest comparison between original and copied executable source;
- makes package boundaries explicit.

Risks:

- this is a large backend structural refactor;
- current repository restrictions limit which backend files may be edited;
- package-private compiler symbols are extensive;
- self-host and fixed-target build scripts depend on the current flat source
  layout;
- moving code can affect compiler size and pruning even without semantic
  changes.

An alternative is to generate the actual compiler package in its final import
location from smaller inputs, but that still leaves a generated package rather
than removing the duplication mechanism.

### Option D: stop transporting the whole compiler as source for every JIT

Longer-term, split preparation into:

- a precompiled, target-neutral compiler kernel component;
- the generated target component;
- a deterministic link step.

Preparation would compile only the generated component and link it to the
precompiled kernel. The release binary would not need a compressed textual copy
of all 33 backend sources.

Benefits:

- removes roughly 414 KiB of compressed source data from the host package;
- reduces preparation time and memory;
- removes exact source-rewrite specialization;
- gives `.rtgb` preparation a clearer component boundary.

Risks:

- Renvo needs a stable component or object-level link contract;
- cross-component reachability and fixed-target pruning become explicit work;
- compiler/kernel ABI versioning becomes more important;
- this may depend on future incremental-linking work.

This is a good destination, but probably not the first cleanup.

### Option E: add a small set of declarative compression features

Add syntax only where at least three definitions repeat the same structural
shape. Good candidates must:

- be facts or bounded templates, not arbitrary control flow;
- specialize to direct Go;
- remain obvious in generated output;
- have a clear validation rule;
- remove enough repetition to justify permanent language surface.

Possible examples:

- instruction families parameterized by width/opcode;
- simple ELF/PE section layouts;
- syscall-number tables bound to one shared local algorithm;
- register-sequence declarations for ABI argument shuffles;
- relocation field layouts.

Avoid:

- expression rewrite systems;
- user-defined macros;
- generic instruction recipes;
- an instruction tape;
- callbacks selected by string at runtime;
- a second untyped expression language.

### Option F: expand the compiler-provided RTG standard library

Move common ELF, PE, runtime, or frame algorithms into versioned RTG API
primitives.

Benefits:

- large reduction in definition size;
- less duplicated and security-sensitive image code;
- easier third-party definitions.

Costs:

- definitions no longer visibly own all behavior;
- the compiler becomes a hidden backend library;
- changing the standard library changes the meaning of old source unless
  strictly versioned;
- custom targets may be forced into built-in assumptions.

This option should be used only for operations that are truly target-neutral.
Byte append, alignment, label, relocation storage, hashing, and bounded scratch
are appropriate examples. A complete PE builder or x86 address encoder is not.

### Option G: replace embedded Go

Replacing embedded Go with another algorithm language would make the files look
less Go-heavy, but would not remove algorithmic complexity. The project would
need to implement and maintain:

- parsing;
- name resolution;
- type checking;
- control flow;
- data structures;
- diagnostics;
- code generation or interpretation;
- debugging support.

The existing Go frontend already supplies those capabilities and emits directly
into the backend package. Unless there is a requirement that third-party
authors cannot write Go, this option has poor cost/benefit.

## 15. Recommended direction

The recommended position for the meeting is:

1. **Keep Go as the embedded algorithm language and generated target.**
   This preserves direct code, uses the existing parser, keeps ordinary Go
   builds working, and avoids inventing another language.
2. **Treat generated Go volume and authored Go dependence as separate metrics.**
   A compact repository is useful, but it is not the same as a compact backend
   definition.
3. **Unify built-in and custom backends on the generated direct-emitter
   topology.** This is the most important correctness and complexity cleanup.
4. **Generate one production projection per architecture.** Do not keep
   contract-only Go that is not authoritative.
5. **After unification, add only a few evidence-based declarative forms.**
   Target a 20–35% reduction in embedded Go, not elimination.
6. **Plan an importable or componentized compiler kernel separately.** That is
   how the 38,473-line compiled copy and the 414 KiB source bundle can
   eventually disappear.

In concrete terms, the realistic outcome is:

- **checked-in generated Go:** potentially reduced by more than 40,000 lines
  after package/component refactoring;
- **embedded definition Go:** plausibly reduced from about 9,725 lines to
  roughly 6,000–8,000 lines with disciplined declarative additions;
- **language implementation Go:** still several thousand lines, because Renvo
  is using its Go frontend to parse/check algorithms and emits Go;
- **runtime host-Go dependence:** remains absent for prepared backend
  compilation and execution.

## 16. Proposed implementation sequence after the meeting

### Step 1: establish one authoritative generated target in tests

Choose one target, preferably `linux/amd64` because it exercises the largest
performance gate and common host path.

- Include its complete generated contract in the actual compiler source set.
- Route it through the generic direct-emitter adapter.
- Remove its old dispatch only after byte/output equivalence and the full
  backend suite pass.
- Measure binary size, compile time, steps, RSS, and self-host behavior.

Exit criterion: one built-in target and the equivalent prepared definition use
the same lowering path.

### Step 2: migrate the remaining built-ins

Migrate by architecture family:

1. 386;
2. AArch64;
3. ARM;
4. VM32/Wasm.

For each family:

- keep the definition as authority;
- merge algorithm and contract projections;
- remove the legacy dispatch branch;
- run native and cross-target equivalence tests;
- preserve fixed-target pruning.

Exit criterion: no built-in calls an architecture-specific scalar compiler
entry that bypasses the generated direct-emitter contract.

### Step 3: delete prepared-mode branching

Once all targets use the stable generated interface:

- remove `renvoPreparedBackend` branches from shared lowering;
- make target specialization select constants and direct bindings at generation
  time;
- delete the inactive kernel if no longer required;
- replace source spelling edits with explicit generation configuration.

Exit criterion: built-in and prepared compilers differ in selected generated
composition, not in shared semantic-lowering control flow.

### Step 4: consolidate checked-in output

- one generated production file per architecture or target family;
- one generated target registry;
- no separate non-authoritative contract projection;
- stale checks regenerate exactly those files.

Exit criterion: every checked-in generated file is either compiled into a
release topology or is clearly a documentation artifact.

### Step 5: reduce repetitive definition Go

Audit normalized repeated roles across at least three definitions. For each
candidate:

1. show the repeated authored code;
2. propose the smallest declarative fact or template;
3. show emitted direct Go;
4. measure net source reduction;
5. reject the syntax if it introduces general-purpose control flow.

Exit criterion: definitions are smaller without acquiring a macro language.

### Step 6: address compiler source duplication

Prototype either:

- an importable backend compiler package; or
- a precompiled kernel plus generated target component.

This should be its own performance-sensitive effort. It changes build topology
more than machine-definition semantics.

## 17. Questions the meeting needs to answer

The meeting should leave with explicit answers to these questions.

### Product intent

1. Is `.rtg` primarily a closed packaging and composition format around Go
   algorithms, or should it be a compact mostly-declarative backend language?
2. Is third-party comfort with the Renvo Go subset acceptable?
3. Is one self-contained source file more important than avoiding repeated
   runtime/format algorithms?

### Source authority

4. Must every generated contract file be checked in, or only the actual
   production projection?
5. Is generated-to-temporary-source testing sufficient for non-production
   projections?
6. May built-in definitions depend on a versioned standard RTG algorithm
   library, or only on primitive emitter operations?

### Production topology

7. Do we agree that built-ins and custom backends should use the same
   direct-emitter path?
8. Can the backend compiler be refactored into an importable internal package,
   despite the current flat standalone compiler layout?
9. Is a component/link model acceptable for JIT preparation, or must every
   prepared backend remain a monolithic compilation from source?

### Performance and review

10. Which metrics are hard gates during unification: generated compiler size,
    output size, runtime, compile steps, RSS, or all existing backend gates?
11. Is checked-in generated line count itself a maintainability gate?
12. Should generated source be optimized for human review, minimum bytes, or a
    balance of both?

### Scope

13. Is reducing embedded Go part of the current PR, or a follow-up after
    unifying production dispatch?
14. Is eliminating `compiler_generated.go` required before merge, or can the
    package-topology refactor follow?
15. Which one target should prove the unified path first?

## 18. Suggested meeting agenda

For a 45-minute meeting:

1. **Five minutes:** agree on the distinction between embedded Go, generated
   Go, and duplicated Go source.
2. **Five minutes:** confirm whether `.rtg` is a packaging format around Go or a
   mostly-declarative language.
3. **Ten minutes:** review the dual built-in/prepared dispatch and decide
   whether unification is mandatory.
4. **Ten minutes:** choose the generated-source topology: one projection,
   importable package, or componentized kernel.
5. **Ten minutes:** choose what belongs in the RTG primitive/standard API versus
   each closed definition.
6. **Five minutes:** assign the first target, performance gates, and merge
   boundaries.

## 19. Bottom line

The current implementation is not "74,000 lines of new backend logic." The pull
request's Go churn includes large generated copies and deletion/replacement of
legacy architecture code. But the concern is still valid:

- the definitions are 83.4% embedded Go;
- the repository carries the backend in several Go representations;
- built-in and prepared targets have not yet converged on one generated
  production path;
- source-copy generation is compensating for the backend's package topology.

We can substantially reduce the visible generated Go and moderately reduce the
Go a backend author writes. We should not try to eliminate Go algorithms by
inventing a second general-purpose backend language.

The most valuable next move is to make one generated direct-emitter contract
authoritative for both built-in and custom targets. Once that is true, redundant
projections and prepared-only branches become removable for structural reasons,
not merely cosmetic ones.
