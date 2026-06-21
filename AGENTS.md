# Retargetable Go (RTG)

rtg is a minimal Golang compiler that compiles a small subset of the Golang language.

The compiler will eventually target many operating systems and architectures. The guiding principle for the compiler is a minimal platform independent core that other more advanced compilers can use as a code generator backend for.

## Restrictions

- The only files that can be modified are `./compiler_*_impl.go` and `./main_rtg.go`.
- You can add new files in `./tests` but you should avoid modifying existing ones unless they are broken.
- The only syscalls are `open`, `close`, `read`, `write`, `chmod`, and `print`.
- Performance requirements are strictly defined in `./main_test.go` and cannot be violated.
- Do not hardcode test cases, emit prebuilt/self-copying binaries, copy the compiler executable or source as the compiled output, or patch the test harness/runtime instead of implementing the compiler from parsed source semantics.
- Your allowed to write your own harnesses and unrestricted tests in `./sandbox` but this folder is not part of the repo and should only be used for local testing.

## Workflow

Every time you find a miscompilation bug you should write a test in `./tests` to confirm it. All tests should only output `PASS\n` if they pass and anything else is considered a failure.

If you want to run tests only use `go test .` since the tests directory will contain conflicting symbols.

## Structure

The compiler is going to be adding more architectures in the future. It's important to keep this in mind and split content between files.

- `compiler_main.go` the compiler entrypoint. avoid putting any code not part of a user interface here.
- `compiler_common.go` any platform independent code.
- `compiler_<arch>_impl.go` any architecture specific code.
- `compiler_<os>_<arch>_impl.go` any operating system specific code.
