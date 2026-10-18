# Managed heap stress and self-host status

## Self-hosting

`TestManagedHeapSelfHost` builds the real `./cmd/renvo` compiler with the bundled
backend and `linux/amd64-desktop` target. The managed stage 1 compiles stage 2,
stage 2 compiles stage 3, and the test requires byte-identical stage-2/stage-3
executables. All stages must enforce a startup environment heap limit.
Stage 3 also compiles a 2 GiB-capped program that runs without reserving that
capacity, and a one-byte-capped program that fails normally at startup. These
exercise the embedded backend's managed range outside the legacy arena range. Stages 1
and 3 also compile and run a 64 KiB program with allocation pressure, a live
buffer and explicit collection.

The compiler budget is 256 MiB and each compilation retains a 60-second deadline.
The existing arena `TestDefinitionCompilerSelfHosts` remains separate. Arena
scratch resets are no-ops under managed collection, persistence keeps identity,
and the managed build does not register the script JIT. Its bundled backend
runs on the native stack; external `.rbe` resolution remains hosted-only.

## Native acceptance stress

`frontend_tests/frontend_managed_heap_stress_test.go` supplies native workloads
run by both `TestManagedHeap` (prepared definition) and
`TestManagedHeapBuiltin` (embedded backend), under the existing 10-second
per-executable deadline:

- interior-only roots spanning page boundaries and a 3 MiB allocation, retained
  through repeated collection, splitting/reuse, and zero-initialization checks;
- 16 seeded cyclic graphs with 96 objects each, transitive references and churn;
- 8,192-node reverse-linked graph, exercising transitive mark-worklist traversal;
- 32 live 256 KiB buffers (8 MiB payload) under a 16 MiB managed limit;
- 384-node graphs surviving repeated collections and allocation pressure;
- 49 active recursive frames holding heap roots;
- heap references retained by deferred arguments and recovered panic values;
- interior strings, pointers into pre-append storage, and zero-length slices with
  nonzero capacity;
- 80 OOM/recovery cycles followed by successful allocation and collection;
- negative and huge allocation sizes, including the signed 64-bit maximum;
- collection during global initialization;
- 1,600 seeded mixed-size allocate/drop operations with complete payload and
  reused-memory zero checks.

The page-boundary test verifies that 4,032 payload bytes fit exactly in a
4,256-byte budget (one 4,096-byte mapping plus 160 fixed bytes), while reducing
the budget by one byte or increasing the payload by one byte fails normally.

These are correctness stress tests, not performance-gate measurements or
production pause-time guarantees. The native collector uses a charged in-segment
page index and an intrusive mark worklist; conservative retention, fragmentation
and full-payload scans can still produce high memory usage or long pauses.

## Independent core model

`internal/managedheap/stress_test.go` adds 32 deterministic seeds with 128 graph
mutation/collection/reallocation rounds each. Logical object IDs and an
independent traversal determine expected reachability. Tests also interrupt root
enumeration and require every allocation to survive that failed collection.

Two edge cases are intentional properties of this nonmoving conservative design:

1. A live middle block separates free holes. A request larger than either hole
   fails at the cap despite sufficient aggregate free space. Removing the anchor
   allows subsequent allocation; the failed attempt does not poison the heap.
2. An integer equal to an interior heap address retains the object. Collection
   cannot distinguish it from a real reference. Removing it permits reclamation.

Both tests require budget compliance; they do not relax limits or demand that
conservative collection behave like a precise/moving collector.

## Definition portability

`TestManagedHeapDefinitionPort` runs the complete native suite with renamed
target, OS, architecture, ABI, runtime and executable-format identities.
`TestManagedHeapDefinitionGranularity` changes mapping units to 8 KiB in the
adapter source only: 8,352 bytes succeeds while 8,351 and 4,256 fail. These
fixtures retain host instructions and syscalls; they are not another OS port.
Contract tests reject missing hooks, wrong signatures, mismatched widths and
unknown ABI versions, while permitting arena definitions without managed hooks.

## Linux QEMU matrix

`TestManagedHeapQEMU` is opt-in with `RENVO_QEMU_MANAGED=1`. It enumerates
`tools/qemu-smoke/targets.json`, requires every emulator, and treats unsupported
managed configurations as failures. It checks pressure reclamation, root
retention, explicit collection, recoverable OOM, compiled caps and environment
overrides. Current result: **33/33 managed targets PASS**, with no missing or
skipped emulators. The separate unchanged arena smoke also passes 33/33. This
is QEMU correctness coverage, not native-hardware or all-target self-host
qualification. See `QEMU.md` for per-target results and reproduction commands.
