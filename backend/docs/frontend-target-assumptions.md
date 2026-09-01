# Frontend target-assumption index

This document indexes production code outside `backend/` which selects compiler
behavior from a built-in target, operating system, architecture, ABI, data
model, or output format. It is the survey for
[issue #447](https://github.com/tinyrange/renvo/issues/447): each item is either
backend-definition behavior to move into RBE, frontend policy to express in a
descriptor, or a host adapter which must remain explicitly platform-specific.

The index records authored sources rather than every generated copy. Tests,
differential-test runners, release tools, flash/debug tools, and REPL terminal
adapters are outside the compiler boundary. They should consume the resulting
registry, but they do not decide how a frontend unit is compiled. Files selected
only because the compiler itself is running on a particular host are listed
separately from target-dependent compilation.

The dispositions used below are:

- **RBE**: target behavior which belongs in the selected backend definition.
- **Descriptor**: generic frontend policy which should query a declared target
  property or capability rather than recognize a name.
- **Policy**: an intentional product default or published-target decision. It
  may remain outside RBE, but it must have one source of truth.
- **Host adapter**: code selected by the platform running Renvo. It is not
  backend behavior, although duplicated target registries should still be
  removed.

## Definition compiler and checked-in backend projections

These are the largest blockers to treating built-in and external RBE backends
uniformly.

| Authored site | Assumption | Disposition |
|---|---|---|
| `internal/rtg/checkedin_host.go`: `GenerateCheckedInTargetProjection` | Dispatches exact target/architecture pairs for Linux, Windows, Darwin, the BSDs, and the Linux kernel. Its fallthrough contains the Linux/amd64 runtime, entry, object, relocation, and ELF image implementation. | RBE |
| `internal/rtg/checkedin_windows_target.go` | Implements and validates the Windows/amd64 PE layout, import ABI, runtime templates, register rewriting, entry code, and compiler-facing names. | RBE |
| `internal/rtg/checkedin_x86_32_target.go` | Implements separate Windows/386 and Linux/386 projections, including PE/ELF constants, runtime operations, entry code, rewrites, relocations, and images. | RBE |
| `internal/rtg/checkedin_aarch64_target.go` | Implements separate Darwin, Windows, and Linux AArch64 projections and their Mach-O, PE, ELF, entry, runtime, register, and relocation details. | RBE |
| `internal/rtg/checkedin_arm_target.go` | Implements the Linux/ARM runtime, EABI ELF image, entry path, and relocation patcher. | RBE |
| `internal/rtg/checkedin_bsd_target.go` | Recognizes FreeBSD, OpenBSD, and NetBSD names and implements shared/divergent syscall ABIs plus OpenBSD/NetBSD ELF layouts. | RBE |
| `internal/rtg/checkedin_kernel_target.go` | Implements the Linux-kernel/amd64 entry/callback ABI, static calls, relocations, and module image adapter. | RBE |
| `internal/rtg/contract_generate.go`: `declarativeArchitectureRelocations`, runtime syscall generation | Whitelists the x86 relative-32, AArch64, and ARM32 relocation algorithms. It also recognizes OpenBSD to record syscall sites for its executable image. | RBE/schema |
| `internal/rtg/format_generate.go` | Contains built-in ELF, PE, Linux-module ELF, OpenBSD, and NetBSD image constructors and fixed relocation-kind lists. | RBE/schema |
| `internal/rtg/resolve.go`: `resolveTarget`, `validDescriptorWidth` | Admits only `native_v1` and `structured32`; restricts widths to 8/16/32/64; defaults all pointer spaces and alignment from the data pointer; and recognizes `wasm32`, `vm32`, `wasm`, `html-wasm`, and `rnvm` when policing the structured family. | Descriptor/schema |
| `internal/rtg/validate.go`: format and target composition validation | Defines ELF/PE as little-endian 32/64-bit formats, knows Linux-module ELF and ARM32 relocation hooks, and imposes the six-word native target composition. | RBE/schema |
| `internal/rtg/generate.go`: `appendPreparedTargetFacts`, `preparedTargetOS` | Converts a closed OS list to backend integer IDs (all unknown OSes collapse to 6), recognizes `sysv_x86_64`, and infers function-symbol support from `rnvm`/Wasm output names. | Descriptor |

The generic parse, import, type-check, identity, artifact, and target-binding
paths in `internal/rbe`, `internal/rtg`, `internal/rtgb`, `internal/backenddef`,
and `internal/unit` do not select a built-in target. They are already the path
that the checked-in projections should converge on.

## Built-in target registry and product policy

| Authored site | Assumption | Disposition |
|---|---|---|
| `internal/targetinfo/targets.json` | Owns the built-in roster, backend/OS/ISA integer IDs, virtual-target aliases, advertised status, IDE status, and release artifact names. The machine facts are subsequently merged from `.rtg` definitions. | Policy; machine facts should come from RBE |
| `internal/targetinfo/cmd/gentargets/main.go`: `frontendSource` | Fixes `linux/amd64` as the default frontend target. | Policy |
| `internal/targetinfo/cmd/gentargets/main.go`: backend registry generation | Selects Linux/amd64 as the authority for the fixed compiler's shared runtime syscall numbers. | RBE |
| Generated `internal/targetinfo/registry_generated.go`, `internal/driver/target_help_generated.go`, `backend/compiler_target_registry_impl.go`, and `backend/compiler_target_policy_impl.go` | Repeat the catalog as target lookup branches, help, IDs, capabilities, build tags, bindings, and defaults. | Generated mirror; change the generator/source, not these files |

`targetinfo.Descriptor` already carries OS, ISA, widths, endian, ABI, image,
build tags, runtime operations, capabilities, virtual status, and arena policy.
Most driver special cases below can therefore be removed without inventing a
second metadata system. Output-mode support, C scalar types/macros, packaging,
and host-run eligibility are the notable missing or underused contracts.

## Driver target selection, modes, and output

| Authored site | Assumption | Disposition |
|---|---|---|
| `internal/driver/options.go`: C option translation | Maps both `-m32` and `-m16` to `linux/386`; the latter adds the code16 flag. The accepted inert GCC flags are predominantly x86/Linux kernel flags. | Descriptor/RBE |
| `internal/driver/options.go`: final option validation | Enumerates the three Windows targets for GUI mode, Linux/amd64 for kernel-module mode, and Linux/amd64 plus Linux/386 for object mode. | Descriptor capabilities |
| `internal/driver/system_profile.go`: `resolveSystemOptions` | Repeats the GUI, kernel-module, and object restrictions. Its object list is narrower than `ParseOptions` and accepts only Linux/amd64. | Descriptor capabilities |
| `internal/driver/backend_definition_host.go`: `resolveBackendBuildOptions` | Correctly uses descriptor OS/capabilities for GUI and kernel modules, but has no generic object/output-mode capability check. | Descriptor; use as the model |
| `internal/driver/browser.go`: `backendTargetForOptions` | Rewrites every kernel-module request to `linux-kernel/amd64`, independently of the selected target. Object and bundle arena defaults are also frontend constants. | Descriptor/RBE |
| `internal/driver/source.go`: `filenameKnownOS`, `filenameKnownArch` | Maintains closed OS/architecture suffix lists so Go filename selection can decide whether a suffix is semantic. Active tags do come from the target registry or external definition. | Descriptor/language policy |
| `internal/driver/compile.go`, `host.go`, `renvo.go`, `renvo_session.go`, `system_output_renvo.go` | Recognize `browser/wasm32` to package Wasm as HTML and to choose non-executable output permissions. The same decision is implemented in five host/self-host output paths. | Descriptor packaging/output kind |
| `internal/driver/diagnostic_full.go`, `diagnostic_renvo.go` | Embed Linux/amd64 in kernel-module/object errors and a RISC-V board name in a board error. These mirror the validation above rather than selecting code. | Generated/generic diagnostic after validation changes |
| `internal/bootstrap/link.go` | The C compiler compatibility linker accepts only Linux/amd64 objects. | Descriptor/RBE or explicitly retained tool policy |

The ordinary build, check, lower, pipeline, link, and unit-binding paths pass a
target name or opaque binding but do not branch on a built-in target. In
particular, `internal/check`, `internal/lower`, and `internal/pipeline` contain
no architecture selection. `internal/link` recognizes language/runtime forms
such as endian helpers and C function pointers, but not an OS or ISA.

## C frontend and machine model

The C path contains both explicit target-name cases and implicit Linux/amd64
defaults. These affect source semantics before a unit reaches a backend.

| Authored site | Assumption | Disposition |
|---|---|---|
| `internal/c11/preprocess.go`: `ppDefaultMacroText` | Starts every preprocessing environment as GCC 5.1, x86-64, Linux, ELF, LP64, 8-bit bytes, and the corresponding scalar typedefs. The driver later undefines/replaces only part of this set. | Descriptor |
| `internal/driver/c_preprocess.go`: `cCompilerTarget` | Parses `os/isa` spelling itself, defaults every unknown ISA to 64-bit pointers, and hard-codes the 32-bit ISA list (`386`, ARM, Wasm/VM32, RISC-V32, Xtensa LX7). | Descriptor |
| `internal/driver/c_preprocess.go`: `cCommandMacros`, `cCommandUndefined`, `cTargetPredefinedMacros` | Defines ILP32/LLP64 scalar macros, special-cases VM32 floating point, knows the x86/ARM/AArch64/Wasm/RISC-V/Xtensa ISA macros, and knows Windows/Darwin/BSD/WASI OS macros and ELF membership. | Descriptor/RBE C profile |
| `internal/driver/c_object.go`: `prepareCSourcesPass` | Chooses ILP32 from the inferred pointer width, LLP64 from the Windows name, and LP64 otherwise. | Descriptor C data model |
| `internal/driver/c_assembly.go`: `CompileCAssemblyCommand` | Uses ILP32 only for the exact `linux/386` spelling and LP64 for every other target. | Descriptor C data model |
| `internal/driver/c_syntax.go` | Both standalone syntax-check paths always use LP64. | Descriptor or explicit syntax-tool policy |
| `internal/driver/c_compiler.go`: `ExecuteCCompilerRequest` | Reports `x86_64-linux-gnu` for `-dumpmachine` and uses Linux/x86-64 macros for compiler probes. | Descriptor or explicit compatibility-tool policy |
| `cmd/renvowasilanguageservice/main_renvo.go`: `analysisCDataModel` | Independently infers LLP64 from two Windows names and ILP32 from architecture suffixes, otherwise falling back to LP64. Its default analysis target is WASI/Wasm32. | Descriptor; share the compiler's C profile |
| `internal/c11/type.go` and the default entry points in `internal/c11/translate.go` | Support only LP64, ILP32, and LLP64, with fixed 8-bit-byte scalar layouts; targetless translation defaults to LP64. | Descriptor/schema |
| `internal/load/package.go` | Defaults C, cgo preambles, and export headers to LP64 when `Source.CDataModel` was not populated; export declarations distinguish only ILP32 from all 64-bit models. | Descriptor plumbing |

Removing target spelling from `cCompilerTarget` is not sufficient: the selected
descriptor (or an explicit C profile carried by the RBE) must reach source
preprocessing, cgo preamble inspection, translation, and assembly metadata.

## Linked-image, preparation, and execution boundary

These sites do not change Go frontend semantics, but they contain duplicate
target tables or constrain which backend a frontend can prepare/run.

| Authored site | Assumption | Disposition |
|---|---|---|
| `internal/linkedimage/image.go`: `targetName` | Converts RNVI integer IDs 1 through 11 to exact target names. It omits the newer BSD IDs even though the registry owns IDs through 14. | Generate from registry/transport descriptor |
| `internal/linkedimage/image.go`, `payload.go`, `native_layout_macho.go`, `native_layout_pe.go`, `linux_layout_*_renvo.go` | Knows the five RNVI format IDs and parses Renvo's little-endian ELF32/64, PE32/PE32+, arm64 Mach-O, Wasm, and VM bytecode subsets. | Format adapter; keep format-specific, remove target-name coupling |
| `internal/backendjit/prepare.go`: `hostTarget` | Duplicates the supported native host table and maps Renvo-hosted preparation to WASI/Wasm32. | Host adapter backed by one generated host-target mapping |
| `internal/backendjit/backend.go`: RTGASM evaluator path | Always compiles and runs assembly evaluators as `vm/vm32` with fixed VM memory/step policy. | Explicit compiler-service target or descriptor capability |
| `internal/rtg/assembly.go`: assembly evaluator contract | Defines the generated evaluator protocol in terms of the ordinary VM32 backend kernel and 32-bit length-prefixed results. | Explicit compiler-service target/schema |
| `internal/driver/run_host.go`: `hostTarget` | Duplicates the native host-to-target table, including Go `arm64` to Renvo `aarch64`. | Host adapter backed by one generated host-target mapping |
| `internal/driver/run_target_*_renvo.go` | Eleven build-tagged files each repeat the self-host target name and RNVI integer ID; the non-native file returns the zero target. | Generate from registry; host adapter |
| `internal/driver/run_renvo.go`, `run_native_*_renvo.go`, `run_session_*_renvo.go` | Requires the RNVI target ID to equal the compiled host and selects Linux ELF, Windows PE, Darwin Mach-O, or BSD native execution implementations by build tags. | Host adapter; target eligibility should be descriptor/transport-driven |
| `internal/runimage` and `internal/backendbridge` platform files | Ordinary-Go JIT execution is limited to listed Linux/Windows/Darwin architectures; the self-host backend bridge has optimized bindings only for Linux amd64/AArch64. | Host adapter |
| `internal/driver/arena_*`, `dirent_*_renvo.go`, and `renvo_path_*_renvo.go` | Select host memory allocation, directory entry, and path ABI implementations. | Host adapter; not RBE migration work |
| `cmd/renvowasi/main_renvo.go` | Defaults the compact self-host frontend to WASI/Wasm32 before accepting an explicit target binding. | Distribution policy |
| `cmd/renvowasibackend/main_renvo.go` | Is an intentionally fixed WASI/Wasm32 backend executable and rejects every other unit/target. | Distribution specialization, not generic frontend policy |
| `cmd/renvowasibackendjit/main.go` | Runs preparation inside VM32, marks only the exact WASI/Wasm32 target runnable, and infers `.exe` output names from a `windows/` prefix. | Descriptor plus compiler-service host target |

## Audit and completion criteria

The survey was produced with three complementary searches: exact target/OS/ISA
names, target descriptor consumers, and implicit width/data-model defaults.
They can be repeated after each migration slice:

```sh
rg -n --glob '*.go' --glob '!**/*_test.go' \
  '(linux/|windows/|darwin/|freebsd/|openbsd/|netbsd/|wasi/|browser/|vm/)' \
  internal cmd
rg -n --glob '*.go' --glob '!**/*_test.go' \
  '(Descriptor\.(Name|OS|ISA|ABI|Image)|runtime\.GO(OS|ARCH)|//go:build)' internal
rg -n --glob '*.go' --glob '!**/*_test.go' \
  '(DataModel(LP64|ILP32|LLP64)|PointerBits|pointerBits|WordBits|Endian)' internal
```

Issue #447's frontend portion is complete when:

1. A built-in RBE uses the same generic preparation/generation path as an
   external RBE; there is no exact-target checked-in projection dispatcher.
2. C preprocessing and translation receive a declared target data model and do
   not infer widths or macros from target spelling.
3. Output modes, virtual packaging, and build tags query descriptors or
   capabilities rather than enumerate built-in names.
4. Target IDs, bindings, help, and host mappings are generated from one catalog;
   generated mirrors contain no independently authored target list.
5. Remaining OS/architecture build tags occur only at genuine host API/JIT
   adapters, and adding an RBE target requires no frontend source edit.
