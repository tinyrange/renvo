# Compiler-core boundary audit

## Scope and result

This audit covers the architecture-independent core migration in PR #589.
The implementation has 379 typed compiler operations, generated from five
bundled integration roots. The core retains source semantics and compilation
ordering; definitions select representation, physical emission, runtime ABI,
and image construction. Prepared backends implement the same semantic boundary
through their direct-emitter adapter.

The structural migration is complete for the existing compiler contracts.
This is not a claim that every possible machine, address representation, or
foreign ABI is implemented. Final acceptance also requires the unchanged
performance/resource gates and full cross-target validation. Current check
results and reviewed revisions belong in the PR, not in this durable source
ownership document.

## Ownership review

| Area | Shared responsibility | Definition or integration responsibility |
| --- | --- | --- |
| Expressions and statements | Parsing, type traversal, evaluation order, reachability, nil/bounds/division semantics | Immediate forms, registers, sized loads/stores, arithmetic, comparisons, branches, peepholes |
| Calls and functions | Signature validation, argument/result traversal, closure discovery, hidden-result planning, helper lifetime | Physical argument/result placement, frame setup, stack/register limits, object wrappers, indirect and imported calls |
| Values and layout | Language kinds, recursive aggregate traversal, normalized value carriers | Native integer and independent address widths, alignment, object byte order, function-address layout, float and split-word policies |
| Runtime operations | Parsed argument evaluation, semantic open flags, helper scheduling, append/growth/copy ordering | Syscall numbers and protocols, register/import sequences, arena capabilities, open-flag bits, discard-page and syscall-site policies |
| Programs and images | Initialization, function queue, speculative closure completion, scratch/cache ownership, bounded session stepping | Entry/frame sequences, object transforms, physical layout, image writer, relocation validation, image-limit diagnostics |
| Source entry | Source loading, parsing, metadata, compile/output handling | Source reserve/release and augmentation policies; prepared single-input adapter |
| Target composition | Contract validation, registry construction, selected definition dispatch | Explicit family, production projection, image variant, runtime and ABI declarations |

The authoritative operation schema is `internal/rtg/compiler_bindings.go`.
Bundled implementations are in `x86_64_compiler.rtg`, `x86_32_compiler.rtg`,
`aarch64_compiler.rtg`, `arm_algorithms.rtg`, and `wasm32.rtg`; shared optimization recipes are in
`lowering_optimizations.rtg`, and shared physical
x86 runtime recipes are in `x86_runtime_intrinsics.rtg`. Generated Go is a
projection, not a second implementation to edit independently.

## Hidden-dependency audit

The review included shared lowering, runtime and linked-image orchestration,
object caching, prepared adapters, production generators, and target metadata.
It did not treat an absence of architecture-name matches as sufficient proof.

- `compiler_common_impl.go` has no concrete architecture/OS/target identity
  decisions, no `code16` or `regParm` decisions, and no prepared-backend identity
  fallback. Emission and representation policies are typed operations.
- Numeric literals in lowering were distinguished from instruction encodings.
  The byte append/patch helpers remain reusable serialization primitives;
  definitions choose when a little-endian or PC-relative-32 recipe applies.
  Shared source lowering no longer chooses raw trap or address opcodes.
- Legacy assembler fields such as `darwinImports`, `openbsdSyscalls`, and
  `wasmLocalSlots` remain storage for backend-owned bookkeeping. Their names do
  not select a lowering path. This migration does not require a new allocator
  or opaque-state object hierarchy just to rename that storage.
- Remaining `renvoFixedTarget == 0` distinctions concern fixed versus embedded
  compiler execution, resource ownership, diagnostics, and optimization scope,
  not a list of output ISAs. The WASI-specific buffer growth in `renvoReadAll`
  is a policy for the running compiler's host memory, not its selected output.
- Target selector constants, public OS metadata, and fixed-target predicate
  evaluation remain in the target integration layer. Bundled dispatch is
  generated from declared selectors; unrecognized emission selectors fail
  rather than silently selecting an ISA.
- Production projection selection is declaration-bound. The BSD image strings
  select explicit format variants; they are not inferred from public target or
  OS names. The remaining `arch.Name == "vm32"` conditions in sequence/direct
  binding generation affect comments only, not emitted executable semantics.
- Registry bounds come from generated tables. The private prepared selector is
  allocated after the registered backend IDs. The driver/context default and
  the validated runtime-number compatibility default are separate roles.
  Kernel policy is capability-derived rather than selected by target identity.

## Width and ABI boundary

The normalized internal value slot remains eight bytes. Strings, slices,
interfaces, split scalars, and aggregate copies retain the compiler's existing
internal representation; a descriptor's pointer width must not silently change
those allocation and traversal strides.

Native object layout is separate. Data, code, and function addresses preserve
their independent descriptor widths. Recursive aggregate layout, native scalar
memory access, function-pointer assignment, and address conversions use the
relevant address space rather than inferring pointer width from language `int`.
Object constants use the descriptor's byte order. Metadata preserves alignment
limits independently of integer width.

Foreign-call layouts such as `sysv_eightbyte` and
`split_register_words8` are explicit, closed protocols, with validated width and
composition restrictions. They do not mean that arbitrary foreign ABIs are
supported. Similarly, preserving a mixed-width or non-flat descriptor is not
proof that every direct emitter can execute every such layout. A new physical
contract still needs its encoder/adapter and execution tests; it does not need
an architecture-name branch in shared lowering.

## Regression evidence

The migration's regression coverage exercises actual parsing, generation,
layout, emission, or system linking rather than a parallel model of the new
policy:

- Unfamiliar bundled selectors, invalid/incomplete bindings, forbidden machine
  fact overrides, safe body projection, labels, local-name capture, recursive
  hooks, shared-tail effects and scope boundaries.
- Renamed production targets across native recipes and rejection of missing,
  unknown, wrong-encoder, or wrong-format projection bindings.
- Independent prepared/registry widths, alignment and byte order; native
  recursive aggregates and scalar pointer/function accesses with an integer
  carrier narrower than the address descriptor; checked address conversions.
- Renamed or missing runtime/ABI policies and malformed/duplicate declarations
  for object calls, split-register calls, open flags, syscall-site metadata,
  directory syscalls, discard pages, and compiler-family composition.
- Native and prepared C-object/aggregate/callback linking, kernel objects,
  VM/WASI IEEE paths, definition-compiler self-hosting, bounded cache/session
  behavior, and focused direct/unit backend corpora.
- Strict generated-source and embedded-bundle parity checks in preflight.

Representative suites are `internal/rtg/compiler_bindings_test.go`,
`internal/rtg/checkedin_projection_test.go`, `internal/rtg/*policy_test.go`,
`internal/rtg/prepared_profile_test.go`, the address/layout tests in
`internal/backendcompiled`, and the registry policy/default-role tests in
`internal/targetinfo/cmd/gentargets`. Execution regressions include
`backend/tests/native_address_scalar_access.go` and
`backend/tests/regression_large_global_initializer_frame.go`.

## Acceptance boundary

No test exclusion, performance baseline, resource ceiling, or CI condition was
relaxed for this migration. Performance acceptance uses
`internal/perfgate/policy.json`; the canonical check driver also enforces its
preflight and full-validation budgets.

The current CI workflow deliberately skips the full backend/frontend, Windows
image, ESP, and macOS package matrix on **all pull-request events**, not only
on drafts. Those jobs run for merge groups and main. A green PR `Required`
check and nine green performance jobs are therefore necessary but do not prove
that the full matrix passed. Marking a draft ready alone does not run it.

Per `AGENTS.md`, ordinary local iteration uses preflight and focused checks;
full local validation requires an explicit request. Merge-queue validation
must be observed to completion, and enqueueing must not be described as a
completed merge. Draft review and authorization to merge remain separate from
implementation completeness.
