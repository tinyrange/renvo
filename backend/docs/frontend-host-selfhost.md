# Host and self-hosted frontend boundary

The host-built frontend and the self-hosted frontend use the same compiler
algorithms and the same backend-facing model. Source parsing, checking, package
building, lowering, linking, and RenvoUnit encoding live in untagged files under:

- `internal/syntax`
- `internal/check`
- `internal/build`
- `internal/lower`
- `internal/link`
- `internal/unit`

`unit.Program` is the shared lowering and linking model. `unit.CoreProgram` is
the smaller canonical boundary between the frontend and every backend;
checker-only and link-only semantic tables are deliberately absent. Both
frontend builds encode that contract with `unit.MarshalCore`.

## Intentional differences

Build-specific code is limited to environment boundaries:

- `internal/driver` uses Go host APIs in stage0 and target runtime calls in
  a self-hosted executable for file discovery, process startup, and backend
  invocation.
- `internal/load` has a target-specific absolute-path adapter where the
  host standard library and the RENVO runtime expose different primitives.
- `internal/backendbridge` selects the native in-process backend bridge by
  operating system and architecture.
- `internal/arena` is harmless on the Go host. The RENVO runtime recognizes
  its intrinsic calls and uses them to reclaim transient compiler storage.
- Packages under `std` may use host-library shims or target runtime calls.
  Their common public API is checked by `std/api_compat_test.go`; graphics,
  filesystem, process, and unsafe implementations remain platform boundaries.
- `renvo_bundle` controls whether standard-library source is embedded in a host
  executable. It changes packaging, not compiler semantics.

The frontend ownership test rejects new build tags in the compiler-core
directories. Structured diagnostic tests and canonical-unit parity tests run
the same workspaces through stage0 and stage3, while the corpus executes the
resulting programs end to end.
