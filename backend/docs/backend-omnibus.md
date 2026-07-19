# Backend omnibus

The canonical staged backend artifact lives in
`frontend_tests/regressions/backend_omnibus`. Its `omnibus` package has no imports,
OS calls, heap requirement, or output dependency. `cmd/app` is only the hosted
adapter that converts a validated result into `PASS\n`.

The exported roots are cumulative:

- `Stage0`: constant return and result-block publication.
- `Stage1`: integer arithmetic, comparisons, branches, shifts, loads, and stores.
- `Stage2`: calls, eight arguments, stack temporaries, and bounded recursion.
- `Stage3` and `RunAll`: globals/BSS, pointers, structs, methods, arrays, and
  aggregate return ABI.

Probe IDs and expected values are literals in the checked source; an emitter
cannot manufacture them. Each probe publishes its ID before executing and
commits its comparison before advancing the two-half 32-bit signature. The
final core-profile signature is `eb10d103ffc30324` after 11 probes.

This is the initial core stage of issue #20. Indirect calls and operations not
yet present in the formal backend contract will extend `RunAll` as their
semantic lowering becomes available. C-reference and constrained-board runs
will use the existing 64-byte result ABI documented in
`docs/omnibus-result-abi.md`.
