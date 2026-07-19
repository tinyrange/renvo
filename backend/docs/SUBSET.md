# Renvo Go Subset

This document describes the direct source language for the backend compiler,
backend tests, and the backend self-hosting compiler. It is deliberately much
smaller than Go.

This is not the full frontend roadmap. The frontend may accept ordinary Go
features outside this direct backend subset by checking and lowering them into
platform-independent units before they reach the backend. The closed frontend
exclusion list is: goroutines, channels, `select`, cgo, and generics. Everything
else, including `defer`, `panic`, `recover`, maps, interfaces, arrays, function
values, dynamic dispatch, complex numbers, ordinary builtins, and unsafe
intrinsics, is frontend work unless this file is updated to make it a backend
requirement too.

## Program Shape

- A program is one or more files in package `main`.
- Tests define one of:
  - `func appMain() int`
  - `func appMain(args []string) int`
  - `func appMain(args []string, env []string) int`
- The runtime provides `main`, calls `appMain` with `os.Args` and optionally
  `os.Environ()`, and exits with its return value.
- Source files may contain top-level `const`, `var`, `type`, and `func`
  declarations.
- Grouped `const (...)` declarations may use `iota` for integer-like enum
  constants.
- Imports are not part of the compiled subset. Runtime functions and constants
  are provided by `backend/renvo_main.go`.

## Compiler Targets

The compiler accepts `-t <target>` for cross-compilation. Currently recognized
targets are:

- `linux/amd64`
- `linux/386`
- `linux/aarch64`
- `linux/arm`
- `windows/amd64`
- `windows/386`
- `wasi/wasm32`

The full test harness runs the targets supported by the current host and
available emulators/runtimes.

## Types

Required:

- `int`
- `int64`
- `byte`
- `bool`
- `string`
- floating point types needed by tests and compiler source
- slices of supported element types, especially `[]byte`, `[]int`, and
  `[]string`
- structs
- pointers to supported types
- named aliases or definitions of supported types
- interfaces and their dynamic type representation

Frontend-in-scope but not directly in the backend subset:

- complex numbers
- maps
- arrays as distinct fixed-length values
- function values and closures
- method values and interface-style dynamic dispatch

Excluded from the frontend and not directly in the backend subset:

- goroutines
- channels
- `select`
- cgo
- generics

## Literals

Required:

- integer literals in decimal form
- integer literals in hexadecimal and binary form
- floating point literals
- character literals for byte-sized values, for example `'a'` and `'\n'`
- interpreted string literals with common escapes such as `\n`, `\"`, and `\\`
- boolean literals `true` and `false`
- `nil` for zero pointer and slice values
- composite literals for structs and supported aggregate values
- keyed and positional struct literals
- empty, nested, and implicit composite literals where the surrounding type is
  known
- slice literals for supported element types, for example `[]byte{1, 2, 3}`
  and `[]int{1, 2, 3}`
- global composite literals and literals using named supported types

Frontend-in-scope but not directly in the backend subset:

- raw string literals
- octal or imaginary literals

## Expressions

Required:

- identifiers
- integer, floating point, string, byte, and bool literals
- parenthesized expressions
- unary `+`, `-`, and `!`
- arithmetic `+`, `-`, `*`, `/`, `%`
- comparisons `==`, `!=`, `<`, `<=`, `>`, `>=`
- boolean `&&` and `||` with Go short-circuit behavior
- bitwise `&`, `|`, `^`, and `&^`
- shifts `<<` and `>>`
- address-of `&x`
- dereference `*p`
- struct field selection, for example `x.y` and `p.y`
- string indexing, for example `s[i]`
- slice indexing and assignment, for example `buf[i] = 65`
- two-bound slicing expressions `x[a:b]`
- full slice expressions `x[a:b:c]`
- slice length with `len(x)`
- slice capacity with `cap(x)`
- function calls
- method calls on concrete receiver values or pointers
- slice append with `append(slice, value)`
- variadic calls to supported variadic functions and methods
- slice expansion in supported variadic calls, for example `append(dst, src...)`
- conversions between supported integer-like types where needed, especially
  `byte(x)` and `int(x)`
- conversion from `string` to `[]byte`
- slice allocation with `make([]T, n)` and `make([]T, n, cap)`
- slice copying with `copy(dst, src)`
- single-result and comma-ok interface type assertions

String concatenation with `+` is optional; tests should avoid requiring it
unless the compiler source needs it.

## Statements

Required:

- `var` declarations with explicit type, initializer, or both
- short variable declarations `:=`, including multiple variables
- assignment `=`, including multiple assignment
- multiple assignment from multi-result calls and ordinary expressions, with
  Go-style right-hand side evaluation before assignment
- compound assignment for arithmetic operators: `+=`, `-=`, `*=`, `/=`, `%=`
- expression statements for function calls and append assignments
- `return` with the number of values required by the function result type
- `if`, `else if`, and `else`
- `switch` statements over supported integer-like, boolean, and string
  expressions, without fallthrough
- `switch` cases with one or more values, optional `default`, `break` that exits
  only the switch, and `continue` when the switch is inside a loop
- type switches with concrete, pointer, interface, nil, multi-type, and default
  clauses, including a correctly narrowed bound variable
- `for` loops in Go's three common forms:
  - `for condition { ... }`
  - `for init; condition; post { ... }`
- `for { ... }`
- `break` and `continue`
- labels and `goto`
- increment and decrement statements `i++` and `i--`

Frontend-in-scope but not directly in the backend subset:

- `defer`
- `panic` and `recover`
- `range`

Excluded from the frontend and not directly in the backend subset:

- goroutines
- channels
- `select`
- cgo
- generics

## Functions

Required:

- named top-level functions
- methods with concrete value or pointer receivers
- zero or more parameters
- zero or more return values
- multiple return values and direct propagation of multi-result calls
- variadic parameters on functions and methods, for example `func emit(xs ...byte)`
- recursion
- calls before declarations

Frontend-in-scope but not directly in the backend subset:

- named return values
- anonymous functions
- method values

## Runtime API

Compiled programs may call only these runtime-provided operations:

- `open(path string, flags int) int`
- `close(fd int) int`
- `read(fd int, buf []byte, off int64) int`
- `write(fd int, buf []byte, off int64) int`
- `chmod(fd int, mode int) int`
- `print(s string)`

Compiled programs may use these runtime constants:

- `O_RDONLY`
- `O_WRONLY`
- `O_RDWR`
- `O_CREATE`
- `O_TRUNC`

## Anti-Cheat Test Guidelines

- Tests should compare compiled behavior against host Go through the existing
  harness.
- Each test file must print exactly `PASS\n` on success.
- A failing test should print a distinct diagnostic and return a non-zero exit
  code.
- Tests should vary source spelling, ordering, whitespace, constants, control
  flow, helper functions, and data values so a compiler cannot pass by matching
  known test text.
- Backend tests should avoid relying on language features outside the direct
  backend subset unless this file is first updated to include them. Frontend
  tests may cover ordinary Go features that lower into backend-compatible units.
