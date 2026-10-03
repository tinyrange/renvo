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
├── *_compiler.rtg              compiler-private generated integration roots
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
matching `*_algorithms.rtg` root and the authoritative production target
projections from their target entrypoints. Architecture roots import only
their shared ISA fragment; target projections use a separate built-in
namespace and call those shared algorithms directly rather than using the
prepared-backend adapter. The second command resolves target descriptors and
writes the registry and the non-enforcing source-volume report.
Architecture-only roots are validated by generation but intentionally
contribute no registry target. The third refreshes the ordinary-Go compiled
backend and the source bundle used to prepare external definitions.

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

Use `go backend` for these portable hooks. A closed checked-in RTG root may also
use `go compiler` for ordinary Go copied only into a checked-in compiler
projection. The x86-32, x86-64, and AArch64 compiler roots use this boundary so
their compiler-private lowering remains RTG-owned without exposing private
compiler types to prepared external backends. Architecture-only compiler roots
have no target identity. A compiler block placed in a target entrypoint is
included in that target's identity even though its source is excluded from
prepared output, preventing a fixed-compiler behavior change from retaining a
stale prepared cache identity. Reserve compiler blocks for integration code
that cannot use the typed `RTGEmitter` surface.

## Bundled compiler bindings

Bundled backends are generated from definitions selected by the `-kernel`
invocation in `generate.go`. The generator does not contain an architecture
allowlist for these bindings. An integration root attaches its selector and
semantic hooks with a separate architecture extension:

```text
extend arch example {
    compiler_selector = exampleSelector
    compiler_bindings {
        copy_primary_to_secondary = exampleCopyPrimaryToSecondary
        # Every operation in the compiler binding contract is required.
    }
}

go compiler {
    func exampleCopyPrimaryToSecondary(a *renvoAsm) {
        // Definition-owned emission and peephole optimization.
    }
}
```

The current migration covers 281 role-based operations: register copies,
pushes/pops, stack slots, immediate values, data/BSS addresses, sized memory
accesses, normalization, arithmetic and logic, comparisons and label branches,
return/frame teardown, split-word immediates, frame comparisons, and the
syscall boundary, scalar multiplication, and relocation finalization. The latter
keeps branch relaxation, layout, and target relocation order in the definitions;
the common PC-relative data displacement helper only performs layout arithmetic.
The 386 normalization peepholes also live with their definition-owned caller.
Local storage units are definition-owned; shared allocation classifies scalar
values, records captured locals, and applies the selected alignment. Cdecl export
and callback wrappers share type classification, argument ordering, and function
reachability in the core; bounded frame, argument, variadic, and private-result
operations own their physical ABI. Reverse-order external register calls are
also definition-owned, without per-call temporary locals in prepared backends.
Process and linked-image entry lowering share one language-level signature check;
definitions receive only validated slice counts and runtime entry-state locations,
then own argument decoding, BSS requirements, and ABI word placement.
Native, prepared, and structured entry paths share global initialization, entry
invocation, and panic handling. Physical incoming frames, reserved-register
restoration, and exit sequences remain bounded definition operations.
Native, VM/WASI, and prepared program setup, function-queue traversal, and
completion are shared,
including bounded incremental compilation. Definitions select physical image
layout, image encoding, object-code transforms, and scratch release lifetimes;
image writers fill an owned result rather than returning another slice copy.
Definitions also select compiler-target fallback semantics, linked-image entry
availability, and object-cache support; cached native targets share the bounded
session path without an architecture-specific scheduler. Prepared application,
object, and kernel outputs use that same lifecycle; speculative closure-label
completion stays in the core and invokes only a bounded empty-function emitter.
Relocation validation and image-limit diagnostics remain fail-closed.
Static-import eligibility is a definition-owned policy query; common call
lowering retains parsed arguments, evaluation order, and portable-body fallback.
Literal parsing selects ordinary or split-word materialization
without inspecting an architecture; each backend owns its scalar-width behavior.
Global-initializer frame setup/teardown and stack IEEE arithmetic, conversions,
comparisons and negation are definition-owned lowering operations as well.
Scalar IEEE conversion, arithmetic and comparison, unsigned division, and the
x86 comparison-flag helpers also live in the definitions; common lowering
retains token interpretation and the language-level division fault check.
Incoming call-word placement and outgoing call ABI emission are typed hooks;
common call lowering retains reachability marking and post-call panic checks.
Platform runtime intrinsics, IRQ stack calls, and Microsoft ABI indirect calls
are definition-owned. The x86 integrations import a shared runtime fragment;
common lowering only recognizes target-neutral string pointers and barriers
before invoking the typed platform hook.
Thread-state register installation/access, stack-runtime calls, unrecoverable
nil-check sequences, and scalar-function ABI emission also use typed hooks.
Common function lowering retains scratch-arena lifetime and metadata cleanup.
The x86 adjacent-push cancellation peepholes live beside their callers.
Local/global word mutation, checked index addresses, external object calls,
register comparisons, and secondary frame-address/dereference emission are
also definition-owned. The core retains expression order, type eligibility,
nil checks, and bounds policy; folded local comparisons are optional target
operations rather than architecture branches.
Assembler reserves and function-symbol requirements are definition-owned as
well. Shared initialization allocates each buffer once from that plan. ELF
symbol/section serialization takes the image writer’s class width explicitly,
rather than deriving record layouts from architecture identity.
Comparison branches, bounded scalar operations and unsigned right shifts,
signed-division guard policy, and bounds/nil-check helper emission are also
bound by definitions. Language-level fault selection and helper lifetime stay
in the core; the 386 shift implementation lives with the x86 integration.
Bulk-copy thresholds, overlap-safe copy emission, zero-helper fast paths,
fresh-arena elision, and local-storage clearing are definition-owned. Common
aggregate lowering retains layout, local allocation, and slice-field semantics.
Split-word arithmetic, unsigned comparison, native wide-operation dispatch,
logical immediate shifts and negation are supplied by the definitions, including
their fixed-target availability policy. Token interpretation and signed-division
fault semantics remain in the common lowering. Indexed-access helper bodies,
reserved bounds/index helpers, slice-check fast paths and aggregate argument
pushing are also definition-owned. Slice-header ABI addresses, append/string
helper selection, tertiary frame stores and fresh-arena copy paths use typed
hooks; concatenation result storage also follows definition-owned allocation/copy
policy. Slice location evaluation and expression semantics remain in common lowering. The
x86 definitions retain their C/code16 peepholes and object-ABI helper implementations rather than placing them in core.
Scalar atom, unary, selector, index, call, and binary evaluation now share a
single language-level path. Definitions select word-constant materialization,
bounded shifts, unsigned comparison result normalization, and ABI intrinsic
eligibility; the core does not select an ISA to evaluate an integer expression.
File open/close/chmod and sequential/offset read/write calls likewise share
argument evaluation while definitions own register placement and runtime entry.
Raw stack and memory instruction encoders are private to their ISA definitions.
Hosted-object calling conventions are selected by definition queries rather
than core architecture predicates. Incoming object register and stack words,
argument-register counts, and export frames are typed operations. Export and
callback wrappers share aggregate-result and variadic lowering; definitions own
private result storage, stack/register callback calls, variadic storage, and
stack-switch helper encodings. Exit, write-value, JIT entry and syscall register
assignment also use definition-owned operations. Arena discard uses a capability
query and a shared page-discard operation rather than architecture tests in
language lowering. The x86-64 global-initializer frame reserves its complete peak
size, including aggregate temporaries beyond the 16-bit ENTER limit.
Foreign static calls use definition bindings for cdecl, register arguments, and
hosted runtime entry. Outgoing memory aggregates use a shared word-location plan
with definition-provided register capacity and aggregate-size policy; the target
runtime owns aligned stack storage, physical copies, relocations, and cleanup.
The same plan is consumed by built-in and prepared SysV definitions. Kernel
module initialization, exit discovery, and callback wrapper lifetimes also share
one lowering path; definitions own entry frames, return sequences, callback
addresses, and foreign call emission. Other
aggregate classification and legacy object ABI assumptions remain migration
work, not completed generic ABI support.
Hooks may take typed parameters, an assembler, compiler-state input, or a
read-only compile-context query, and a validated result type. Context queries
return an explicit unavailable value for an unknown selector; emission operations
still mark an unknown selector as an emission failure. Their complete
signatures are checked before generation;
missing operations, duplicate selectors, and unknown operations are errors.
The generated dispatcher projects definition-owned bodies directly into their
selected branches to avoid another call at each emission site. Identical tails
after leading calls and scoped `if` statements share code, including when all
matching bodies have a prefix. Declarations, assignments, labels and other
control flow stop splitting, so prefix-local names cannot escape into a tail.
Prefix selection remains exclusive even when a prefix changes the context, and
terminal returns alone are not shared. It caches the
context pointer to avoid repeated nested loads while retaining direct fact reads
for fixed-target branch elimination, without duplicating selection conditions.
The cache carries the same non-null context invariant as compiler construction.
Generated local names cannot capture identifiers supplied by a definition. Hooks with
noncanonical parameter names or function-scoped labels retain direct calls.
Private projected entrypoints are omitted from the architecture source unless
another binding, Go body, or definition declaration still references them.
An unrecognized emission selector fails compilation rather than falling back to an ISA.
Prepared backends provide the same compiler operation names through their direct
emitter and ABI bindings, without depending on bundled hooks or selectors.

Compiler-binding extensions cannot override the imported machine's facts.
Bundled compiler profiles project data, code, and function pointer widths,
maximum alignment, endianness, and default arena sizes from the resolved
descriptors rather than inferring them from architecture identities. Targets
with non-default arena limits declare `arena_default` explicitly; this preserves
the existing compiler limits when using the descriptor projection. Bundled and
prepared profiles also derive their six runtime-operation bits and hosted bit
from the same definition facts; a familiar target identity does not imply
filesystem operations or hosted execution.

This contract is a migration boundary, not a claim that all compiler-private
coupling has been removed: other layout/ABI rules, calls, remaining emitter
operations, runtime composition, and fixed-target orchestration still contain legacy architecture
knowledge. Those must be migrated before a bundled definition list alone can
control the compiler's complete target set.

Windows/386 uses the same bounded runtime sequences for prepared and fixed
compilers. Its checked-in projection adds only the compiler-facing names and
runtime-helper cache state needed by the shared compiler. The resulting fixed
compiler is intentionally allowed a narrow 324 KiB size budget so those
readable sequences do not need a second compact byte-template implementation.

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

### Runtime syscall policy

A runtime may opt into page reclamation with a `discard_pages` block containing
exactly `page_size`, `number`, and `advice` integer fields. `page_size` must be a
positive power of two no larger than 1 GiB; the syscall number and advice must be
nonnegative. The runtime must provide a syscall number register, at least three
argument registers (address, byte count, advice), and an instruction. Omission
disables reclamation, regardless of the target's OS or ABI name. Prepared lowering
aligns the requested range inward to complete pages and uses these declared
parameters; invalid or duplicate policies fail preparation before emitting code.

A runtime `syscall` block may declare `site_table = address_number_pairs`.
The prepared runtime adapter then records each syscall instruction offset and
operation number for the image writer's syscall table. Absence means no table;
unknown layouts fail prepared validation. The policy is independent of the
public target and OS names. OpenBSD's runtime declares this layout explicitly.

### Foreign object-call classification

An ABI may declare `object_call_layout = sysv_eightbyte` to opt into the
existing 64-bit SysV eightbyte aggregate-classification protocol (up to two
register words). This is a closed protocol identifier, not the ABI declaration's
name; absent, duplicate, malformed, or unknown policies do not infer support
from a familiar name. The runtime must independently provide `emit_static_call`
for outbound object calls, and kernel-module composition does not enable that
path. Prepared generation rejects this layout with non-64-bit words. Display
names for the ABI, target, and OS have no effect on this policy.
