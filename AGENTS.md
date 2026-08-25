# Renvo

Renvo is a minimal, retargetable compiler for a practical subset of Go. The
frontend lives at the repository root and the native code generators live in
`backend/`.

## Reference Guide

Refer to `GUIDE.md` for practical background, workflows, debugging techniques,
and architectural context gathered from working on the project. It is a
non-binding reference: `AGENTS.md`, current tests, and current implementation
take precedence whenever they disagree with the guide.

## Restrictions

- Backend compiler edits are limited to `backend/compiler_*_impl.go`,
  `backend/compiler_main.go`, and files under `backend/definitions/`.
- Frontend acceptance tests live under `frontend_tests/`.
- New backend regression programs may be added to `backend/tests/`; avoid
  modifying existing programs unless they are broken.
- Do not modify `backend/main_test.go` or any
  `backend/*performance*_test.go` file except for explicitly authorized
  harness refactoring. Never loosen performance or resource constraints without
  explicit user instruction.
- Backend compiler implementation may invoke only `open`, `close`, `read`,
  `write`, `chmod`, and `print`. This restriction does not limit target runtime
  operations or APIs implemented by the Renvo standard library.
- Performance requirements are strictly defined in `backend/main_test.go` and
  cannot be violated.
- Do not hardcode test cases, emit prebuilt or self-copying binaries, copy the
  compiler executable or source as compiled output, or patch the harness/runtime
  instead of implementing parsed source semantics.
- One-off experiments may be written under the ignored `sandbox/` directory.

## Workflow

Every miscompilation bug needs a minimal reproducer in `backend/tests/`. A
passing regression prints only `PASS\n`.

When debugging a backend bug:

1. Reduce the failure to a minimal source program.
2. Build a stage0 compiler and reproduce the generated-output failure there.
3. Use GDB and temporary diagnostics as needed.
4. Fix the compiler from parsed source semantics.
5. Add the reproducer to `backend/tests/` and run the relevant compiler tests.

Do not use `go test` in module mode inside `backend/tests/`; those standalone
programs intentionally contain conflicting package-level symbols. Do not use
`go test ./...` as a whole-repository check because it descends into independent
corpus modules and local scratch programs. Prefer explicit package sets and
`go test ./frontend_tests` plus the intended `./backend` harness tests.

Use the repository-owned check driver for presubmit work:

```sh
./tools/check preflight
./tools/check full
```

The first command is the normal development loop: it catches stale generated
sources, package failures, and test compilation errors and enforces a one-minute
budget. Do not run `full` during normal interactive agent work: it can consume
the complete 30-minute timeout and populate several gigabytes of Go and Renvo
test cache. The merge queue runs that validation. Use the narrow modes and
focused corpus filters while iterating, for example:

```sh
./tools/check backend 'tagged_memory_access'
./tools/check frontend 'map_frontend_lowering'
```

`full` adds the backend, every compiler performance/resource gate, and the
complete frontend suite including self-hosting. Only run it locally when the
user explicitly requests the full validation. On a systemd-based Linux host, set
`RENVO_CHECK_MEMORY_MAX=4G` to isolate each command from the interactive
session. Do not change check modes to skip or weaken a gate; use a narrower
documented mode only while diagnosing a failure.

Positive backend programs have a sibling `.expected` file, and positive
frontend modules have an `expected.txt` file. Normal corpus checks compare
Renvo output directly with these checked-in values and must not invoke host Go
as an oracle. After adding a positive test, run
`go run ./cmd/renvoexpect -write` and review the companion expectation.

## Frontend scope

The exclusion list is closed: generics and cgo are out of scope for now.
Goroutines, channels, and `select` may be accepted only as frontend syntax that
is lowered completely through the runtime handler ABI before the compact unit
reaches a backend; they remain outside the direct backend source subset. Every
other ordinary Go feature is frontend work unless the project explicitly
changes that policy. This includes defer, panic/recover, maps, interfaces,
arrays, function values, dynamic dispatch, complex numbers, ordinary builtins,
and unsafe intrinsics.

## Backend structure

- `backend/compiler_main.go`: compiler entrypoint and command interface.
- `backend/compiler_common_impl.go`: platform-independent compiler code.
- `backend/compiler_<arch>_impl.go`: architecture-specific code.
- `backend/compiler_<os>_<arch>_impl.go`: operating-system and architecture
  integration.
