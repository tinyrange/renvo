# Retargetable Go (RTG)

rtg is a minimal Golang compiler that compiles a small subset of the Golang language.

The compiler will eventually target many operating systems and architectures. The guiding principle for the compiler is a minimal platform independent core that other more advanced compilers can use as a code generator backend for.

## Restrictions

- Backend compiler edits are limited to `./compiler_*_impl.go` and `./compiler_main.go`.
- Any edits under `./rtg/` are allowed.
- You can add new regression programs in `./tests` but you should avoid modifying existing ones unless they are broken.
- Do not modify `./main_test.go`.
- The only syscalls are `open`, `close`, `read`, `write`, `chmod`, and `print`.
- Performance requirements are strictly defined in `./main_test.go` and cannot be violated.
- Do not hardcode test cases, emit prebuilt/self-copying binaries, copy the compiler executable or source as the compiled output, or patch the test harness/runtime instead of implementing the compiler from parsed source semantics.
- You are allowed to write one-off custom tests in `./sandbox`, but this folder is not part of the repo and should only be used for local compiler experiments. Do not use it to modify or replace the test harness.

## Workflow

Every time you find a miscompilation bug you should write a test in `./tests` to confirm it. All tests should only output `PASS\n` if they pass and anything else is considered a failure.

There are no restrictions on the specific `go test` command you run.
Do not run `go test` in module mode inside `./tests`; those files intentionally contain conflicting package-level symbols and are meant to be compiled individually by the compiler test harness.
Do not treat `go test ./...` as a valid whole-repo check. It descends into `./tests` and `./sandbox`, which intentionally contain standalone programs with conflicting symbols and scratch experiments. Prefer explicit package sets such as `go test ./rtg/...` plus the top-level compiler harness tests you intend to exercise.

## Bugfixing Workflow

Whenever a compiler bug is encountered:

- Make a minimal reproducer for the bug.
- Compile a stage0 compiler and debug the reproducer with that stage0 compiler until the generated output works there.
- Use GDB freely to inspect compiler and output-binary failures.
- Add compiler features that make debugging easier when useful, such as symbols or better diagnostics.
- Once the minimal reproducer is understood and fixed, add it to `./tests` as a regression program.
- Run the compiler tests again and repeat the loop until the compiler works end to end.

Debug prints and debug helpers are allowed in the main repo while bugfixing. Keep or remove them later based on what is useful.

## Structure

The compiler is going to be adding more architectures in the future. It's important to keep this in mind and split content between files.

- `compiler_main.go` the compiler entrypoint. avoid putting any code not part of a user interface here.
- `compiler_common.go` any platform independent code.
- `compiler_<arch>_impl.go` any architecture specific code.
- `compiler_<os>_<arch>_impl.go` any operating system specific code.
