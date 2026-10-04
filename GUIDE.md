# Working with Renvo

This is the practical guide I wish existed before doing substantial work on
Renvo. It complements `README.md`, `AGENTS.md`, and the specifications under
`backend/docs/`. Those files define the project; this file concentrates on the
mental models, workflows, traps, and debugging techniques that have proved
useful in real development.

Renvo is pre-1.0 and moves quickly. When prose and executable tests disagree,
trust the tests and current implementation, then fix the stale prose.
The repository module path is `renvo.dev`, and the codebase is licensed under
Apache-2.0.

## The short version

Renvo is three related things:

1. A normal-looking Go frontend that loads modules, checks a practical Go
   subset, and lowers a linked program into a versioned Renvo unit.
2. A very small self-hosting backend that consumes either its direct source
   subset or a lowered unit and writes ELF, PE, Mach-O, or Wasm itself.
3. A bootstrap and retargeting project whose constraints—small compiler,
   bounded memory, no target linker, and minimal target testing—are product
   requirements rather than incidental optimizations.

The most important working rules are:

- Do not confuse the full frontend scope with the smaller direct backend
  source subset.
- Reproduce a miscompile with a stage0 backend before changing it.
- Every backend miscompile gets a minimal `backend/tests/` regression that
  prints exactly `PASS\n`.
- Do not run `go test ./...`; this repository intentionally contains many
  independent programs and modules.
- Do not modify `backend/main_test.go` or weaken a performance gate to make a
  change pass.
- Track compiler size, CPU, and RSS on fixed workloads; investigate regressions
  and pathological outliers even when the frontend has no temporary hard gate.
- Test Windows behavior on native Windows and Darwin behavior on native
  macOS. Wine, QEMU, and format parsers are valuable but not sufficient.
- Keep the frontend/backend handoff deterministic. Host-built and self-hosted
  compilers must describe the same program.

## Build a development toolchain

A Go-built bootstrap invokes a sibling backend. Actual `renvo` frontends are
Renvo-built artifacts and always link the backend in-process:

```sh
mkdir -p sandbox/bin
go build -o sandbox/bin/renvo-backend ./backend
go build -tags renvo_bundle -o sandbox/bin/renvo-bootstrap ./cmd/renvobootstrap

sandbox/bin/renvo-bootstrap \
  -t linux/amd64 -s -o sandbox/bin/hello ./path/to/hello
```

The bootstrap resolves `renvo-backend` beside itself. Pass
`-bootstrap-backend <path>` immediately after the executable name only when a
test or packaging workflow deliberately stores the backend elsewhere.

`renvo_bundle` embeds `std/`, `forms/`, and `device/`. It does not change
compiler semantics. Without bundling, set `RENVO_STDROOT` or arrange an
adjacent source tree.

Running the bootstrap without arguments is intentionally useful:

```sh
sandbox/bin/renvo-bootstrap
sandbox/bin/renvo-bootstrap --help
```

The command requires `-o` for compilation. The default target is
`linux/amd64`, but development and tests should normally pass `-t` explicitly
so host defaults cannot hide a target mistake.

### Build a genuinely standalone compiler

To produce the one-file self-hosted toolchain:

```sh
sandbox/bin/renvo-bootstrap \
  -tags renvo_bundle \
  -t linux/amd64 \
  -s \
  -o sandbox/bin/renvo-standalone \
  ./cmd/renvo
```

Move `renvo-standalone` to an empty directory and compile a small module there.
That isolation test matters: it catches accidental dependencies on the
checkout, `std/`, the Go toolchain, an adjacent backend, or the original
executable path.

A standalone compiler contains:

- the frontend;
- the supported standard-library source bundle;
- an in-process backend;
- the target writers needed for all supported outputs.

The release workflow builds fully bundled Go-hosted compilers on native Linux,
Windows, and macOS runners. Each artifact embeds the standard library and
built-in targets while retaining dynamic RTG/RBE preparation for Tier 2
backends. The workflow verifies those capabilities, stages a draft release,
uploads every raw binary, and publishes only after all three artifacts exist.

## Everyday command-line use

Compile a package:

```sh
renvo -t linux/amd64 -s -o app ./cmd/app
```

Compile exactly named files:

```sh
renvo -t windows/amd64 -o app.exe main.go platform.go
```

Explicit files must be `.go` or `.c` files in one directory and package.
Exactly those files are used. For the explicit root file list, build
constraints and OS/architecture suffixes are ignored and test files are
skipped.
Dependencies still use normal target selection rules. Prefer package mode when
you want ordinary Go build selection.

Useful options include:

- `-t <target>`: choose the output target;
- `-s`: strip optional symbols and metadata;
- `-tags a,b`: add build tags;
- `-arena-size <bytes>`: set the generated program's arena limit;
- `-system <file.rtg>`: load a hosted target and hard binary/arena limits;
- `-windows-gui`: choose the Windows GUI subsystem;
- `-emit-unit`: stop after frontend linking and write the canonical unit;
- `-emit-image`: write the `RNVI` linked-image transport;
- `-mode=kernel-module`: build a Linux/amd64 kernel module;
- `cc -c`: compile one standalone C translation unit to a system-linkable
  Linux/amd64 ELF relocatable object;
- `-I <dir>` / `-I<dir>` / `-isystem <dir>`: add C header search roots;
- `-script`: treat one explicit file as a script.

### Hosted system profiles

A hosted system profile keeps the constraints used for a compiler or
application build beside the project:

```text
system "small-linux-amd64" {
    target = "linux/amd64"
    binary = 2MiB
    arena = 32MiB
}
```

Use it instead of separate `-t` and `-arena-size` options:

```sh
renvo -system systems/small-linux-amd64.rtg -o app ./cmd/app
```

`binary` limits the final executable bytes, including browser HTML packaging
when applicable. The frontend rejects an oversized result before writing the
output. `arena` is passed to the backend as the generated program's hard
managed-allocation limit. It is not a claim about total process RSS, which also
includes stacks, code pages, loader mappings, and operating-system overhead.

Profiles accept byte counts with `B`, `KiB`, `MiB`, or `GiB` suffixes. Both
limits and the target are required. `-system` cannot be combined with `-t` or
`-arena-size`.

The frontend size checks report two payloads independently: the stripped
compiler with its native backends against the former 2,000,000-byte reference,
and the offline bundle containing `std/`, `forms/`, and `device/` against the
former 4 MiB reference. Those historical comparisons are telemetry; both payloads
must fit the shared 8 MiB ceiling. The checked-in
`systems/frontend-linux-amd64.rtg` profile applies that full-bundle limit and
gives the running compiler a 192 MiB arena:

```sh
renvo -system systems/frontend-linux-amd64.rtg -tags renvo_bundle -s \
  -o renvo ./cmd/renvo
```

`-emit-unit` and `-emit-image` are intentionally different:

- a unit is the frontend/backend semantic handoff;
- an image is already backend-generated native or Wasm content wrapped in a
  format-neutral transport for inspection or in-process loading.

Use `-emit-unit` to isolate frontend and backend behavior:

```sh
renvo -emit-unit -o sandbox/program.unit ./cmd/app
renvo -emit-unit -o - ./cmd/app > sandbox/program.unit
```

If host and self-hosted frontends produce different canonical units, the bug is
above the machine-code emitter. If the units agree but executables differ, the
bug is in backend lowering, target emission, image composition, or runtime
integration.

### Multi-target embedded programs

A Go executable can ask the frontend to compile another entrypoint for a
different target during the same invocation:

```go
//renvo:compile -t linux/386 helperMain
var helperELF []byte

//renvo:compile -t freestanding/amd64 longModeMain
var longModeEntry uintptr
```

The directive must immediately precede one uninitialized package variable and
has the exact form `//renvo:compile -t <target> <entry>`. The entry must be a
top-level function in the root package. A `[]byte` variable receives the whole
foreign executable as static program data; treat that slice as immutable. A
`uintptr` variable receives the address of an in-place executable entrypoint,
and is accepted only when the target advertises `in_place_entry`.

Each directive creates a separately target-bound child unit. Reachability is
computed from its selected entrypoint, so a function called by both the native
entrypoint and a foreign entrypoint is lowered independently for both
architectures. `-emit-unit` preserves the unresolved child units; a normal
compile sends every child to its target backend, embeds the resulting artifact,
then compiles the parent. One external RTG definition may therefore export all
targets involved in a multi-stage image.

The `examples/bios-longmode` program demonstrates a `bios/8086` parent building
page tables and transferring to an embedded `freestanding/amd64` child.

## Targets and output policy

The frontend currently recognizes:

| Target | Normal output |
| --- | --- |
| `linux/amd64` | static PIE ELF |
| `linux/386` | static PIE ELF |
| `linux/aarch64` | static PIE ELF |
| `linux/arm` | static PIE ELF |
| `windows/amd64` | PE |
| `windows/386` | PE |
| `windows/arm64` | PE |
| `darwin/arm64` | PIE Mach-O |
| `freebsd/amd64` | static PIE ELF |
| `openbsd/amd64` | static PIE ELF with pinned syscall sites |
| `netbsd/amd64` | static ELF |
| `wasi/wasm32` | WASI WebAssembly |
| `browser/wasm32` | browser HTML containing WebAssembly |
| `vm/vm32` | deterministic Renvo bytecode with 32-bit words and pointers |

Hosted native images separate executable and writable storage:

- Linux has distinct RX and RW loads.
- Windows has RX text, RW data, and NX compatibility. It currently has no
  base-relocation directory, so do not advertise `DYNAMIC_BASE`.
- Darwin has `__TEXT`, `__DATA`, and `__LINKEDIT`, and uses ARM64
  position-relative references suitable for dyld sliding.
- Wasm keeps mutable state in linear memory and code in the engine.
- VM bytecode runs with caller-supplied instruction and linear-memory limits.
- `vm/vm64` is reserved for a future 64-bit-word VM target.

Avoid relying only on an old architecture document for these details. The
format tests in `backend/compiler_executable_layout_test.go` are the executable
source of truth.

`-mode=kernel-module` is currently restricted to `linux/amd64`. A source file
may provide the module license with:

```go
// renvo:module-license Dual MIT/GPL
```

Conflicting or invalid license directives are frontend diagnostics, not values
to guess in the backend.

## Modules, packages, build tags, and embedding

Renvo looks upward from the working directory for `go.mod`. It supports local
packages, target-specific files, build constraints, local `replace` directives,
vendored dependencies, and a read-only module cache.

Renvo never downloads dependencies. Resolve them through one of:

1. a local `replace`;
2. the module's `vendor` tree;
3. a pre-populated `RENVO_MODCACHE`.

This is deliberate. A standalone compiler must remain useful offline and must
not acquire an undeclared network dependency.

The `renvo` build tag is implicit. OS, architecture, `unix`, `wasi`/`wasip1`,
and `browser` tags are derived from `-t`. User tags are added with `-tags`.
The `renvo_bundle` tag selects packaging code and also gives a compiler-sized
default arena; do not use it casually on ordinary applications.

`//go:embed` is handled by the loader before semantic lowering. Supported uses
include strings, byte slices, `embed.FS`, and glob patterns. When changing
embedding:

- enforce module-root boundaries;
- preserve deterministic path order;
- test both host and self-hosted filesystems;
- test a bundled compiler away from the repository;
- avoid regenerating giant Go byte-literal files—the standard bundle itself
  uses `//go:embed`.

## Language scope

Renvo is not a drop-in Go compiler.

The alternate C11 frontend under `internal/c11` feeds the same checked-package
and linked-unit pipeline as Go without invoking a host C toolchain. Mixed `.go`
and `.c` files use a cgo-style explicit boundary: a preamble immediately before
`import "C"` declares that Go file's C functions, while `//export` functions
are published to C through a synthetic `_cgo_export.h`. The `#pragma go`
C-project workflow remains a C-first direct-binding mode. Keep
C-specific parsing and source adaptation above the shared checker boundary; do
not duplicate the linker or add C semantics to a target backend. The supported
C subset and its current limitations, including the initial integer-only export
header ABI, are documented beside that frontend.

Generics are frontend work. The frontend checks constraints and specializes
generic declarations into concrete types and functions before ordinary lowering.
Backends receive the existing concrete source subset. Generic checking and
specialization live in `internal/check/generic_*.go`; acceptance cases include
`generic_builtin_identity`, which checks inferred universe types when package
types hide their names. Specialization adds a declaration-only package of
builtin aliases only when needed; original package indices remain stable.
`generic_alias_scope` checks receiver, parameter, result, and captured names
that hide concrete types in function bodies and closure signatures, along with
local type declarations and function-value conversions. Body-inserted type
text uses package aliases when needed to keep its checked lexical identity.
Graphs containing generic declarations also check ordinary function bodies
before specialization, including unused functions in imported packages.
Array lengths and constant bounds use the selected target's integer width,
including when a 32-bit frontend compiles for a 64-bit target. Array identity
stores lengths in 64 bits; explicit literal keys must fit target `int`, while
inferred positions and lengths can extend beyond its maximum. Target-width
tests compare declaration acceptance with Go's checker for both widths. A VM32-hosted frontend also compiles and executes a 64-bit program containing
huge zero-element arrays and slices and function-size constant array bounds.
Signed VM comparisons handle different-sign operands before subtracting, since
the VM comparison flag stores a wrapped 32-bit difference. Peephole load markers
identify the instruction kind as well as its register, so literal and displacement
bytes cannot be mistaken for neighboring instructions.
`generic_ordinary_bodies` covers concrete parameters, methods, closures, and
indexing untyped string literals and constants. String indexing produces a byte
value, so it cannot be used as a constant initializer.
The generic checker models the existing `open`, `read`, `write`, and `chmod`
runtime primitives, including their integer results and argument types. The
legacy `close(int) -> int` form remains available for file descriptors; a type
parameter constrained by `~int` does not become a valid channel-close operand.
`generic_runtime_os` checks file I/O through generic function values and the
Renvo standard library with process arguments supplied on both native and VM
targets.
Function-value storage includes an unused zero-length array of slices to retain
Go noncomparability through boxing and containing arrays or structs. Comparisons
through interfaces and use as dynamic map keys still panic, including typed nil
functions and empty containers. `generic_function_interfaces` covers these
cases, cross-package function aliases, assertions, methods, and pointer controls.
Defined function types keep separate declarations and interface identities from
anonymous function types. Matching signatures share their implementation tags,
so assignments and conversions retain callable closures. Ordinary function
values use a fixed descriptor containing a tag and an interface holding a typed
environment pointer. Method environments snapshot the receiver at binding;
closure environments retain captured variables. Dispatch and native defer
registration restore each concrete environment type.
Tuple assignments through native function values obtain their result type from
the callable signature before looking up a named declaration. This preserves
interface result fields and local callbacks that shadow global functions.
`generic_tuple_callbacks` covers inferred and explicitly typed callbacks across
packages.
Generic map type replacements preserve enclosing parentheses, including the
call delimiters in `make(map[K]V)`. Parenthesized type arguments and declared
map types are resolved before map construction and indexing are lowered;
`generic_map_types` exercises creation, nil conversions, pointers, named types,
and aliases. Function-range checking resolves named and constrained yield
function types, including variadic yields, while retaining the required builtin
`bool` result identity. Integer and function ranges require Go 1.22 and 1.23,
respectively, including in unused generic definitions.
Function ranges lower to typed callbacks before function-value conversion in
both whole-program and incremental links. The lowering evaluates the iterator
once, creates fresh declared bindings, preserves assignment destinations and
nested branches, and propagates outward branches and returns after the iterator
finishes. Named results are assigned before iterator cleanup. Yield state checks
detect resumed, exhausted, reentrant, and swallowed-panic invocations.
Interface iterator methods retain their declared signatures, including embedded
and aliased interfaces. Parentheses around named yield types are normalized
before the compact backend boundary.
`generic_range_functions` and the upstream slices/maps iterator fixture cover
these paths. Defers in an iterator body retain the enclosing function as their
owner. Typed invocation records snapshot callees and arguments into one list
per invocation, including defers outside the loop so their LIFO order is retained.
Compiler-generated invocation and dispatch frames explicitly forward the owner's
recovery permission; ordinary authored helpers retain normal recover semantics.
`generic_range_defers` covers normal returns, named results, nested owners,
methods, builtins, nil callbacks, and nested/replaced panics. Variadic yield
signatures are checked and execution has been validated on AMD64, x86-32, ARM,
and AArch64. Go 1.25.5's compiler crashes on the separate variadic-yield probe,
so that probe remains outside the host-Go expectation corpus.
Deferred tagged function values select and register their actual callable in
the original function's defer stack. Callees and arguments are captured once;
direct recovery, concrete/interface method values, tuple arguments and variadic
snapshots are covered by `generic_defer_callbacks`. Interface method values
check nil receivers when selected, before deferred arguments are evaluated;
interfaces containing typed nil pointers retain their concrete method behavior.
Discarded `recover()` calls lower to blank assignments in their original frame,
including parenthesized calls, simple control-statement initializers, and
for-clause post statements.
A defer executed on a
helper's normal return cannot recover a panic already active when that helper
was entered; the native defer ABI grants recovery only for a new panic unwound
by its owner.
The same acceptance module checks defined and anonymous assertions, generic
function aliases, assignments, returns, and conversions between defined types.
`generic_function_conversions` also checks anonymous function-type conversions,
typed nil conversions, single evaluation of captured callbacks, aliases inside
container signatures, and nested function returns. Function-type aliases are
resolved in their lexical scope before storage selection; defined parameter
types retain their identity. Nested function results receive concrete storage
and chained calls dispatch in source evaluation order.
`generic_nested_callbacks` checks higher-order callbacks across packages,
parameter names inside nested function types, native callback parameters,
callback literals in slices, arrays and maps, captures, and direct calls through
function type assertions. Parameter/result lists and assertion parentheses are
type syntax rather than function conversions. Container literals supply the
element's declared function type when choosing callable storage. Import removal
uses declaration boundaries so explicit semicolons and shared source lines do
not remove neighboring declarations.
`generic_aggregate_callbacks` checks aggregate aliases in callback signatures,
including imported structs and interfaces and private types from different
packages. Matching aggregate alias bodies share a signature spelling without
moving their declarations; aliases with private members retain their package
identity. The return checker parses a closure's full signature before skipping
its body, including braces belonging to parameter and result types. Interface
method declarations sharing a scalar type's name are not conversions.
Builtin bridges preserve primitive and builtin `error` identity across
package boundaries, boxing, generic methods, and closure results. Other cases include
`generic_functions` and an unchanged copy of Go 1.25.5's `cmp` package in
`generic_upstream_cmp`, plus upstream slice algorithms in
`generic_upstream_slices` (only dependency import paths are adapted). Runtime compatibility with the upstream Go standard
library is a separate task.
`generic_intrinsic_aliases` checks authored callback and global names that
match runtime intrinsic aliases, including names with existing suffixes and
generic specialization prefixes. Linked intrinsic aliases avoid authored names.
When an intrinsic alias changes, a `//renvo:intrinsic` declaration marker keeps
its backend ABI identity; unmarked functions with similar names stay ordinary
calls. The compact unit format is unchanged, and package-artifact linking retains
these markers.

A method selected through a type parameter must retain Go's receiver semantics
after specialization. Method expressions become concrete forwarding functions;
method values retain the outer receiver through a method-only interface. This
matters when an embedded interface field changes after binding the method value.
`generic_promoted_methods` covers this distinction, private cross-package
methods, nested embedding, interface assertions, and variadic method calls and
expressions. `generic_embedded_pointer_paths` covers successive embedded pointer
dereferences mixed with value embedding, reads and writes through slice/call/
assertion bases, captured variables, nil pointer recovery, and direct methods
that hide promoted methods. It also covers range loops inside closure conditions.
`generic_unsafe` covers unsafe pointer conversions, cross-package
specializations of unsafe pointer types, and package declarations that shadow
predeclared value names. Type-set interface declarations are checked even when
the source has no generic declarations or instantiations; their use as value
types is rejected in signatures, variables, conversions, assertions, literals,
and builtin type operands, including unused functions and closures. Aliases and
imports retain this distinction, while local values and types may shadow a
constraint name. `generic_constraint_values` covers valid value-interface
conversions and shadowing through both ordinary and generic functions.
`generic_predeclared_shadow` covers package callbacks named after predeclared
types and builtins, local alias and value shadowing, initializer visibility,
and type parameters that shadow package values. Type positions reject visible
values even when they have a predeclared type's spelling.

`//renvo:reflect` on a generic named struct applies to its concrete
specializations. Compiler-owned `//renvo:typename` comments retain semantic type
and embedded-field names through source lowering and frontend caches. Embedded
alias bridges have one declaration per field identity across the package graph,
so package symbol disambiguation cannot disconnect their selectors or literal
keys. `generic_reflection` covers names, alias fields, private-field visibility,
field reads and writes, nil pointers, and grouped reflection annotations using
Renvo's existing reflection API. The predeclared `error` type retains its defined
interface identity; it is distinct from `interface { Error() string }`, and a
pointer to it has no methods.

Runtime identity compares anonymous interfaces by their complete method sets,
including embedded methods, signatures, and variadic arguments. Private struct
fields and interface methods include their declaring package in identity.
Compact package ownership must survive transient linking, and hoisted anonymous
type aliases retain their source package. Unicode identifier lowering preserves
export status. `generic_type_identity` exercises these rules across packages,
including pointers, arrays, slices, function signatures, and Unicode members.

Dot-imported names retain their declaring package in checked name and type
references. Linking can then apply the same alias to declarations and uses when
another package or local binding requires renaming. Aggregate field and method
names remain member declarations rather than package references.
`generic_dot_import_identity` covers ordinary and generic calls, aliases,
signature types, imported variables, constant array lengths, callbacks, and
anonymous fields across normal and dot-importing files in the same package.

Function initializers use the same callable representation through transparent
parentheses, both for inferred locals and globals. Named function conversions
wrap native implementations in that type's shared callable storage. Resolve
initializer operands before introducing new local bindings, including when a
local function variable shadows the function supplying its initial value.
`generic_function_initializers` covers these cases alongside grouped declarations,
reassignment, reused short declarations, nil conversions, imported methods, and
parenthesized calls returning function values.

Grouped `var (...)` declarations retain each specification's own initializer
scope, including multi-result calls and closures containing declarations.
Linking expands the group into ordinary `var` declarations before callable
lowering, preserving declaration and initialization order. The
`generic_variable_groups` acceptance module covers local and global groups,
initializer shadowing, callbacks, tuple results, and generic function bodies.

Unsafe layout operands are checked as values even though they are not evaluated.
Untyped operands must fit their default type. `Offsetof` requires one accessible
field and rejects methods, ambiguity, and promotion through embedded pointers;
an explicit pointer base is allowed. Variable-size checking propagates through
arrays and structs, but stops at pointers, slices, maps, channels, interfaces,
and functions. This also determines whether an enclosing array `len` or `cap`
can be constant. `generic_unsafe_layout` covers these rules and shallower field
selection. The generic array-length checker uses its expression evaluator when
the ordinary wide-constant evaluator cannot handle the expression.
Scalar `unsafe.Sizeof` results are constants during generic checking, using the
destination word width and Renvo's scalar storage. They can supply array lengths
without losing the resulting array's type identity. Backend constant evaluation
shares layout queries between package constants, local constants, and array
bounds; queries never evaluate their operands or cache unresolved layouts.
Numeric constant checking also covers fixed containers and aggregate offsets.
The frontend layout model uses the selected target's language alignment and
lowering ABI: maps and slices use descriptor storage, ordinary pointers occupy
normalized slots, and object output uses native aggregate and pointer layouts.
Target alignment and object mode participate in package-cache identities.
Ordinary function-value descriptors occupy 24 bytes in the normalized ABI;
object function values remain native code pointers. Numeric function layout
constants therefore work before collecting implementations. Queries use an
explicit callable operand type so direct function names and local callbacks
agree, and never evaluate their operands. `generic_callable_layout` checks
constant array bounds, cross-package aliases and identity, bound receiver
snapshots, closures, containing aggregates, and nil values.
Empty structs, zero-length arrays, and arrays of empty
elements have zero value size and stride. The backend keeps their addressable
scratch storage separate from data copies and unsafe queries. Zero-element
slices retain logical length/capacity without proportional backing allocation;
expanded append updates their descriptor without per-element copying.
`generic_zero_layout` checks generic array identity, empty fields, callbacks,
bound receivers, defer, boxed values, maps, slices and array materialization.

Generic constant evaluation keeps exact rational values until a concrete type
supplies the rounding context. Specializations emit rounded floating and complex
constants as exact hexadecimal binary literals, including constants passed to
calls, composites, and builtins. `generic_constants` covers these contexts and
float32 double rounding. Map types in specialized bodies need concrete spelling
because their lowering helpers live at package scope, outside local parameter
aliases.

At the ordinary numeric boundary, a typed real value combined with untyped
imaginary zero keeps its real type and defined name. The backend uses that type
for storage and comparison instructions. Leading decimal literals end a source
line like other numbers; their first character is not a continuation operator.
The compact-unit reader accepts a separately encoded decimal point and remaps
token ranges after recovering the literal. Numeric regressions cover both raw
source and unit input, including VM32 and 32-bit native targets.

Generic language checks use the loader's effective version for each source file.
Type parameters and instantiation require Go 1.18, the concrete comparable
exception requires Go 1.20, function-value and interface-argument inference use
Go 1.21 rules, and generic alias declarations require Go 1.24. A declaration and
its caller can select different versions, including through leading build
constraints; the instantiation's file owns its constraint-satisfaction rules.

The command driver passes destination word and pointer widths through
`pipeline.Config` to `load.Graph.Layout`. Generic constant representability,
unsigned complement, array lengths, and constant indices use that layout.
Resolved custom backend descriptors supply the same widths. Keep the layout in
each build session and its package-cache identity so concurrent and incremental
builds cannot reuse a different target's checking context.

Goroutines, channels, and `select` are accepted by the frontend and lowered to
the `renvo.dev/x/runtime` handler ABI before backend compilation. Version 1
handlers serialize Renvo execution; this is concurrency, not parallelism. The
direct backend source subset still does not accept those constructs.

Invalid generic definitions and instantiations must fail in the frontend with
structured diagnostics, including definitions that are never instantiated.

Every other ordinary Go feature is frontend work unless the project explicitly
changes the policy. That includes:

- arrays and slices;
- maps;
- interfaces and dynamic dispatch;
- function and method values;
- closures;
- `defer`, `panic`, and `recover`;
- named and multiple returns;
- type assertions and type switches;
- complex numbers;
- ordinary builtins such as `min`, `max`, and `clear`;
- unsafe intrinsics.

The backend also has a direct source language used by its tests and by the
self-hosted backend compiler. That subset is deliberately smaller. Features
accepted by the full frontend may be converted into simpler unit operations
before the backend sees them. Read `backend/docs/SUBSET.md` when writing a
direct backend test and do not use a frontend-only feature accidentally.

## The standard library

Renvo ships a focused standard library under `std/`, currently including:

```text
bytes       embed       encoding    errors      fmt
graphics    io          os          path        process
sort        strconv     strings     testing     unicode
unsafe
```

User programs import familiar names such as `"fmt"` and `"os"`. Repository
packages use the `renvo.dev` module path, for example `renvo.dev/forms` and
`renvo.dev/std/graphics`.

It is intentionally smaller than Go's standard library. API compatibility is
improving, but package presence does not imply complete Go behavior. Check the
actual package and tests before promising an API.

There are usually two implementations:

- host code used while developing with Go;
- `renvo` code compiled into target programs.

Their shared API must remain compatible. `std/api_compat_test.go` catches many
signature differences, but semantic tests are still needed. The same named
symbol should behave the same in host and target implementations.

Identity matters for sentinel values. For example, a `bytes.Buffer` must return
the canonical `io.EOF`, not a private error with the same text. Interface
dispatch, equality, and `errors`-style behavior can distinguish identity from
spelling.

## Scripts

Script mode is opt-in and intentionally small:

```go
import "fmt"

answer := 2 + 2
fmt.Println(answer)
```

```sh
renvo run script.go
renvo run script.go -- first second
```

The script command supplies `package main` and `func main()` around top-level
statements. It accepts one file and targets the native host. Keep the wrapper
as a frontend transformation; the backend should continue to compile ordinary
linked programs.

Self-hosted native `run` uses a linked-image entry ABI. It maps the image,
applies final segment protections, builds Renvo argument and environment
values, switches to a suitable native stack, and calls the entry point. It does
not use a temporary executable as a substitute for the linked-image ABI.

## The REPL

Build the pure Renvo REPL with a standalone compiler:

```sh
renvo-standalone \
  -tags renvo_bundle \
  -t linux/amd64 \
  -s \
  -o renvorepl \
  ./cmd/renvorepl

./renvorepl
```

The REPL is real incremental linking, not source replay:

- accepted statements are never executed again;
- each successful submission becomes a new linked-image generation;
- stable symbol cells preserve variables and pointers;
- earlier code generations stay executable so saved function values and
  closures remain valid;
- declarations, imports, and assignments extend the live session.

Useful commands are:

- `:help`;
- `:history`;
- `:source`;
- `:reset`;
- `:quit`.

The line editor is implemented in Renvo and supports cursor movement, history,
UTF-8-aware redraw, and tab completion. Completion and signature help reuse the
shared parser/checker/language-service infrastructure. Do not build a second
identifier parser inside the REPL. Import completion should query the bundled
standard-library packages, and argument help should use checked function
signatures.

When testing the REPL, pipe a sequence of submissions rather than checking only
one expression. A useful session covers:

1. an expression;
2. a variable declaration and mutation;
3. a closure that captures the variable;
4. a saved function value;
5. an import;
6. a selector call after the import;
7. another call through the old saved function.

That sequence catches accidental source replay, broken stable cells, stale code
pointers, and import-generation mistakes.

## Forms

The Forms designer treats Go source as the model. A typical project separates:

- `main_form.go`: user-owned event handlers and logic;
- `main_form_generated.go`: designer-owned generated control construction.

Do not edit the generated file by hand or make the designer rewrite the user
file. Generated assets such as fonts and control icons should be embedded with
`//go:embed`, not copied into giant source literals.

Forms has ordinary widget/model unit tests and UI automation and accessibility
tests that exercise real focus, control identity, actions, and screen state.

Keep visual automation deterministic. Prefer control identity and accessibility
state over pixel coordinates. Incremental compilation must invalidate by
semantic inputs—source, selected target, tags, dependencies, and generated
form—not merely by file timestamp.

## Compiler architecture

The frontend pipeline is roughly:

```text
filesystem/module discovery
        ↓
source selection + go:embed expansion
        ↓
syntax parsing
        ↓
package graph loading
        ↓
semantic checking
        ↓
platform-independent lowering
        ↓
package linking + canonical symbols/init order
        ↓
Renvo unit v1
        ↓
direct backend emitter
        ↓
ELF / PE / Mach-O / Wasm or RNVI
```

The important frontend packages are:

- `internal/syntax`: scanner and parser;
- `internal/load`: module and package model;
- `internal/check`: semantic checking;
- `internal/build`: build orchestration;
- `internal/lower`: frontend feature lowering;
- `internal/link`: package linking, symbols, and initialization;
- `internal/unit`: in-memory unit model;
- `internal/driver`: command, filesystem, bundle, and execution boundaries;
- `internal/backendbridge`: in-process backend boundary;
- `internal/linkedimage`: `RNVI` decoding and native layout;
- `internal/repl`: persistent-generation source and completion state.

The compiler-core algorithms live in untagged files so host and self-hosted
frontends execute the same logic. Build-tagged differences belong at true
environment boundaries: filesystem access, process APIs, memory mapping,
backend invocation, and target standard-library adapters.

An ownership test rejects new build tags in core frontend directories. If an
algorithm seems to require separate host and target versions, first look for a
smaller environment interface.

## Units and linked images

The unit format is a versioned, length-delimited semantic handoff. Its
authoritative schema is:

```text
backend/unit/schema.json
```

The generated human-readable reference is:

```text
backend/docs/unit-v1.md
```

Fixed-width fields are little-endian. Counted semantic tables use compact
varints and token spans. Unknown child records can be skipped by length, while
duplicate known records and missing required records are rejected.

Keep units deterministic:

- sort filesystem entries before loading;
- make package, function, type, and symbol ordering explicit;
- never depend on Go map iteration;
- do not let traversal order assign persistent symbol identities;
- compare host and self-hosted canonical units in tests.

`RNVI` is not a second IR. It wraps already generated target content and records
the target, native format, checksum, segments, relative entry point, and native
payload. Loaders must validate all lengths, offsets, permissions, and address
ranges before mapping anything.

## Backend model and ABI invariants

Renvo currently uses a direct-emitter IR. Shared lowering calls typed storage,
value, control-flow, and call operations, and the selected backend emits bytes
as operations arrive. There is no materialized instruction graph yet.

The three transient value locations are:

- primary: scalar result or pointer;
- secondary: second operand, string length, or slice length;
- tertiary: third operand, index, or slice capacity.

Unless explicitly documented, an operation may clobber all three. Store a
value in a compiler local, global, or temporary stack slot before emitting an
operation that must not destroy it.

The normalized backend storage slot is eight bytes. This is not proof that the
target pointer or language `int` is eight bytes.

Array metadata keeps the logical length in `arrayLength` (`uint64`), independently
of host-sized compiler counts and storage sizes. Inferred lengths use the all-ones
sentinel until the literal is resolved. A 32-bit compiler targeting 64-bit code
must preserve both words of a parsed array bound, type identity, and emitted
`len`/`cap` values. Storage-size multiplication is checked before narrowing;
an unrepresentable array size is marked negative. Struct metadata propagates
that marker and checks field-size addition and alignment before narrowing.
Such a type can describe a pointee and supply a constant length, but requesting
physical storage still fails. This does not establish support for materializing
gigantic aggregates or implementing all of Go's zero-size storage rules.

Important stored layouts are:

- string: pointer, length;
- slice: pointer, length, capacity;
- interface: payload, canonical dynamic-type tag;
- struct: declaration-order fields at normalized aligned offsets;
- tuple: aggregate layout with a hidden result pointer when required.

Interfaces are two words. A nil interface has two zero words. An interface
holding a nil `*T` has a zero payload and a nonzero `*T` tag and therefore is
not equal to `nil`.

Interface calls must dispatch from the declared contract and runtime type tag,
not from source spelling or the first compatible implementation found while
walking a function. Register pointer/value receiver variants before emitting
dispatch tables, and keep candidate order deterministic.

Arguments are flattened left to right into Renvo call words. Calls between
Renvo functions use the Renvo ABI, not the platform C ABI. Operating-system and
foreign calls require explicit adapters.

Do not leak physical-register assumptions into common lowering. Adding an
architecture means mapping the Renvo value locations, fast call words,
overflow words, stack slots, control flow, and relocation operations to that
architecture.

## Runtime and arena behavior

Renvo uses bounded arenas rather than a general garbage collector. Current
backend defaults are:

| Class | Default |
| --- | ---: |
| 64-bit hosted program | 128 MiB |
| 32-bit hosted program | 64 MiB |
| WASI program | 32 MiB |
| Linux kernel module | 64 KiB |
| compiler built with `renvo_bundle` | 256 MiB |

`-arena-size` accepts 256 bytes through 1 GiB. A reservation is virtual address
space, not necessarily resident memory, but it still matters under address
space limits and on 32-bit hosts.

Arena exhaustion must enter the normal panic path. It must be recoverable by
`defer`/`recover`, and an uncaught OOM must print a useful panic and exit
nonzero. Never discover OOM by touching an unmapped guard page.

The self-hosted frontend also uses arenas for compiler memory. The practical
rules are:

- Use `arena.Mark`/`Reset` around transient compiler phases.
- Copy anything that must outlive a reset into persistent storage.
- Release complete unused pages with `arena.Discard`/`DiscardBytes`.
- A slice's allocation may begin after an alignment gap. Compute discard
  ranges from the actual backing allocation, not merely the pre-allocation
  mark.
- Capacity growth in a persistent slice can pin all scratch allocations made
  after a function mark. Preallocate persistent tables before entering
  per-function scratch regions.
- After copying a growing source buffer, discard complete pages from the old
  buffer and from the unused tail of the new buffer.
- Do not retain token, source, or diagnostic slices owned by a reset
  compilation attempt. Reparse or persist the small required value.

RSS bugs are often ownership bugs, not “the compiler needs more memory.”

## Platform lessons

### Linux

- Validate PIE behavior by actually observing randomized mappings, not only by
  checking `ET_DYN`.
- ELF `PT_LOAD` file offsets and virtual addresses must be congruent modulo
  alignment.
- Keep RX and RW loads separate.
- Linked-image execution should map segments with their final permissions and
  use an isolated native stack.
- QEMU is excellent for cross-architecture regression coverage, but still run
  native amd64 and ARM64 jobs because emulators can tolerate ABI mistakes.

### Windows

- PE parsing and Wine execution are not enough. Native Windows has exposed
  failures that Wine accepted.
- Win64 requires shadow space and correct 16-byte stack alignment at imported
  calls.
- Expression evaluation can reach an import call at either stack parity.
  Dynamically align a fresh call area, preserve the exact original stack
  pointer, and restore it after the call.
- A misaligned Win64 call may run hundreds of times before a path inside
  `ntdll` uses `movaps` and crashes. The crash can look like a filesystem bug
  because it occurs under `CreateFileA`.
- ARM64 and x64 need a realistic growable PE stack reservation.
- Keep `.text` non-writable and data non-executable; do not claim ASLR without
  emitting a relocation directory.
- When native CI crashes, rerun under GDB and capture registers, stack words,
  the caller instructions, and API arguments. On Windows, put multiword GDB
  commands in a command file—command-line quoting can silently reduce
  `info registers` to `info`.

### Darwin/ARM64

- Executable JIT mappings require `MAP_JIT` and
  `pthread_jit_write_protect_np`.
- macOS rejects some `MAP_FIXED | MAP_JIT` combinations. Reserve the overall
  address range, unmap the executable subrange, then request the JIT mapping at
  the exact address and verify that the returned address matches.
- Keep JIT pages under the platform's write-protection protocol rather than
  relying on an ordinary `mprotect` sequence.
- Flush or synchronize instruction visibility as required by ARM64.
- Test on a real macOS runner. A Mach-O parser cannot validate VM policy, dyld
  behavior, or JIT entitlements.

### Wasm

- WASI and browser output share much lowering but have different packaging and
  host contracts.
- `browser/wasm32` wraps the Wasm output as browser HTML.
- Linear-memory sizing follows arena policy.
- Keep the complete WASI self-hosting compiler within the shared policy described
  in `docs/performance.md`.

## Correctness areas that deserve disproportionate testing

Some bugs pass simple examples and fail only when values cross a storage or ABI
boundary. Always test these forms separately:

- a constant expression;
- assignment to a local;
- assignment to a global;
- a function argument;
- a function return;
- an interface box/unbox;
- storage in a struct, array, slice, or closure.

This is especially important for:

- 64-bit shifts, arithmetic, multiplication, and division on 32-bit targets;
- signed extension and truncation;
- multiple and named returns;
- aggregate hidden-result pointers;
- deferred updates to named results;
- closure capture by shared variable identity;
- typed nil interfaces;
- value versus pointer receiver dispatch;
- failed single-result and comma-ok assertions;
- type-switch binding and source-order matching;
- slice and string element bounds;
- array and slice expressions nested under selectors or indexes;
- `bytes`/`io` sentinel error identity.

Native closure dispatch must include compatible literals before their factories
are emitted. Constructing a handle establishes its capture layout and requeues
any body deferred earlier. At queue exhaustion, referenced literals whose
constructors were never emitted receive fault labels without compiling dead
bodies. `closure_transitive_factory` and `closure_dispatch_unreachable` cover
these cases. The function queue can grow when a body is requeued; its capacity
must participate in scratch-arena lifetime tracking, or a self-hosted compiler
can retain a queue allocation after rewinding the memory that owns it.

Runtime faults must use the same panic machinery as explicit `panic`:

- nil dereference;
- division by zero;
- slice/string index or slice bounds;
- failed type assertion;
- arena exhaustion.

A recovered fault must continue correctly. An uncaught fault must report the
panic and exit nonzero rather than becoming SIGSEGV, SIGFPE, SIGILL, or status
zero.

## Testing without stepping on traps

Do not run:

```sh
go test ./...
```

`backend/tests/` contains standalone programs with conflicting declarations.
The frontend corpus contains thousands of independent modules. `sandbox/`
contains scratch experiments. A recursive module test is not a meaningful
whole-repository check.

The canonical entry point is `tools/check`. It keeps the test partition in one
reviewable place shared by developers and CI:

```sh
./tools/check preflight
./tools/check backend
./tools/check performance
./tools/check frontend
./tools/check full
```

`preflight` is the normal development loop. It checks generated-source
authority first, then the tracked package and bundled-build suites, and finally
compiles the backend and frontend test packages. It has a hard one-minute
budget so stale generation or an ordinary Go build failure is always cheap to
discover. `full` runs every mode in that order, can consume its complete
30-minute timeout, and provides merge-queue-equivalent local validation. Do not
launch it as an interactive development check unless full local validation was
explicitly requested. The narrower modes are diagnostic tools, not a substitute
for the queue checks before a change is merged.

Compiler fuzzing, self-hosting, and wide backend runs can consume enough memory
to kill an interactive session when they regress. On systemd-based Linux, use:

```sh
RENVO_CHECK_MEMORY_MAX=4G ./tools/check full
```

The driver places each child command in a memory-limited user unit and returns
its real status. This is an outer safety boundary only; it does not replace or
relax Renvo's measured compiler RSS limits.

Useful package checks are:

```sh
go test ./internal/... ./std/... ./cmd/...
go test ./backend/unit ./backend/target ./backend/bringup ./backend/omnibus/... ./backend/cmd/...
go test ./frontend_tests
```

Useful backend harness checks are:

```sh
go test -run '^(TestCompileTests|TestUnitFrontendCompileTests)$' ./backend
```

For one backend or frontend regression, prefer the repository driver's focused
filter. It is a regular expression matched against slash-normalized corpus case
names and prints periodic progress plus the five slowest cases:

```sh
./tools/check backend 'tagged_memory_access'
./tools/check frontend 'map_frontend_lowering'
```

The ordinary backend suite and the compiler performance driver are separate.
Use `./tools/check backend` for backend tests and `./tools/check performance`
for the selected target's complete compiler self-hosting gate.

The frontend corpus has several layers:

- `quick/`: 300 tests intended for every frontend check;
- `extended/`: 2250 broader cases;
- `regressions/`: permanent hand-maintained cases;
- `negative/`: exact rejection diagnostics;
- `std_compat/`: standard-library compatibility checks.

The frontend package runs the complete corpus, bundled frontend, and
self-hosting paths by default:

```sh
go test -count=1 ./frontend_tests
```

To test a particular self-hosted frontend:

```sh
RENVO_FRONTEND=/absolute/path/to/renvo \
  go test -count=1 ./frontend_tests
```

Positive backend programs use sibling `.expected` files and positive frontend
modules use `expected.txt`. Corpus execution compares the Renvo program directly
with these checked-in values instead of building every program with host Go.
Run `go run ./cmd/renvoexpect -write` after adding a positive case, then review
the generated expectation. Frontend negative tests retain their exact
`expect.json` diagnostics.

The backend test cache lives under `backend/.renvo/test-cache`. Its keys include
compiler sources and test inputs. Preserve the cache during ordinary work; if
you find an invalidation bug, fix the key rather than disabling caching.

### Reading a failed run

Start with the first failed check mode and its first concrete diagnostic. One
compiler defect can otherwise produce a long tail of stage1, stage2, corpus,
and self-hosting failures that are not independent bugs.

- A generated-source digest or RTG authority failure is drift: regenerate the
  named authority and review the resulting diff before running anything wider.
- A Go build or package failure belongs to the host implementation and should
  be fixed before interpreting backend output.
- Exit 137, `signal: killed`, or a vanished terminal usually indicates the
  outer memory boundary, separate from the compiler's measured memory gate. Reproduce the
  named child command inside `RENVO_CHECK_MEMORY_MAX` and inspect that first.
- A gate failure should name runtime, RSS, binary size, or frontend CPU. Do not
  trade one budget for another or raise a threshold to make the aggregate green.
- Wrong output, a target crash, or a stage mismatch is a compiler bug. Reduce it
  to the smallest applicable `backend/tests/` or `frontend_tests/regressions/`
  case before editing compiler code.

When a rerun passes without a source or environment change, record it as a
flaky or infrastructure failure rather than treating the green rerun as a fix.

### Regression style

For a backend miscompile:

1. add one minimal program under `backend/tests/`;
2. make success print only `PASS\n`;
3. make each failure path print a distinct clue and return nonzero;
4. vary values and control flow enough that source-text matching cannot pass;
5. run both direct-source and unit-frontend paths where relevant.

Frontend regressions normally belong in a separate module under
`frontend_tests/regressions/`. Negative cases should assert phase, diagnostic
code, location, and message.

Do not “fix” a test by changing an existing regression to match broken
behavior. Add a focused new regression unless the old test itself is genuinely
wrong.

## Performance gates are architecture constraints

Native and WASI Tier 1 targets use the bundled frontend/backend compiler to build
itself. VM32 measures a prepared custom backend compiling a semantic regression
program from a compact unit.
The shared policy lives in `internal/perfgate/policy.json`: 8 MiB compiler,
256 MiB peak memory, and median growth limits of 25% CPU, 20% memory,
10% artifact size, and 20% VM instructions against a pinned source reference.
Both revisions execute on the same runner. Larger increases block feature
inclusion for maintainer evaluation case by case.

```sh
./tools/check performance
RENVO_PERF_TARGET=vm/vm32 ./tools/check performance
```

`ci-performance` enforces exactly the same policy. Separate Linux, Windows, macOS, and virtual-target workflows run all nine
Tier 1 targets on PRs, merge-queue commits, and main.
See [performance.md](docs/performance.md) for workload, measurement definitions,
reference handling, and limitations. CPU time is user plus kernel CPU; wall time
is telemetry. Memory is peak RSS on Linux/macOS and peak job commit on Windows.

When a gate moves, diagnose the phase:

1. source discovery/read;
2. parse/check;
3. lower/link/unit encoding;
4. backend parsing and metadata;
5. per-function emission;
6. final image construction.

Track current RSS, high-water RSS, arena marks, slice length/capacity, and final
code/data/relocation counts. Remove noisy diagnostics after the cause is
understood, but keep generally useful failure detail.

Common size wins are:

- use compact fixed instruction templates for invariant sequences;
- avoid generic machinery in the single-target compiler;
- pre-size persistent tables from known counts;
- use zero-capacity branches when a feature is absent;
- keep target-specific helpers out of unrelated backend builds.

Common RSS wins are:

- reset scratch at function/package boundaries;
- discard complete dead pages promptly;
- stop persistent slices from growing inside scratch regions;
- avoid retaining oversized read capacity;
- encode and release package state incrementally.

Never add target-name or source-text special cases to satisfy a gate. The
compiler must still derive output from parsed semantics.

## A reliable bug-fixing workflow

1. **Reduce the failure.** Put the smallest useful experiment in ignored
   `sandbox/`.
2. **Classify the layer.** Compare host behavior, frontend unit, backend output,
   and runtime failure.
3. **Build stage0.** Reproduce with a host-built backend before relying on a
   possibly broken self-hosted compiler.
4. **Inspect the artifact.** Use `readelf`, `objdump`, `dumpbin`, `otool`,
   `wasm-tools`, or format parsers as appropriate.
5. **Debug the output.** Use GDB freely. Disassemble around the failing PC and
   inspect the actual ABI values.
6. **Fix parsed semantics.** Do not patch the harness, copy prebuilt bytes, or
   special-case the test.
7. **Add the permanent regression.**
8. **Run focused tests.**
9. **Run the resource/performance gate affected by the change.**
10. **Run native CI for platform-sensitive work.**

Compiler crashes and generated-program crashes are different:

- a compiler crash usually points to malformed ownership, indexing, or missing
  diagnostics in the frontend/backend;
- a generated-program crash usually points to lowering, ABI, memory layout,
  runtime checks, or image policy.

For backend failures, useful temporary capabilities are welcome: symbol maps,
relocation dumps, deterministic labels, phase counters, and clearer
diagnostics. Keep them when they improve future debugging without causing an
unexplained payload regression.

## Frontend development advice

- Reject unsupported syntax during parsing, loading, or checking with a
  specific diagnostic. `RENVO-BACKEND-001` should not be the first useful
  message for a source error.
- Evaluate side-effecting operands once. This matters for assignments,
  selectors, indexing, assertions, switches, and deferred calls.
- Preserve Go evaluation order when lowering multiple assignment and calls.
- Lower a feature into explicit unit semantics rather than teaching each
  architecture a high-level Go construct.
- Keep package initialization order canonical, including blank imports.
- Preserve declared symbol identity across packages, aliases, methods, and
  interface contracts.
- Use the same checked model for interactive tooling.
- Keep build tags out of parser/checker/linker algorithms.
- Compare stage0 and self-hosted units whenever a change touches canonical
  ordering or arena reclamation.

When adding a builtin, implement its type rules in the checker and its semantic
lowering in the frontend. Do not merely recognize its name in a machine-code
emitter.

## Backend development advice

Backend compiler edits are restricted by `AGENTS.md` to
`backend/compiler_*_impl.go` and `backend/compiler_main.go`. Do not modify
`backend/main_test.go`.

The file split is intentional:

- `compiler_common_impl.go`: platform-independent lowering/emission support;
- `compiler_<arch>_impl.go`: instruction encoding and architecture ABI;
- `compiler_<os>_<arch>_impl.go`: OS/architecture integration;
- image/OS helpers: headers, imports, syscalls, and entry adapters;
- `compiler_main.go`: command interface.

Before adding an operation to common code, ask:

- Is this a language semantic, a Renvo ABI operation, an ISA operation, or an
  executable-format policy?
- Does it assume pointer width, `int` width, byte order, register names, page
  size, or a hosted OS?
- Can the Wasm and future C backends implement the same contract?

Keep relocations explicit and reject unresolved or out-of-range references.
Silent truncation produces binaries that are much harder to debug than a
compiler diagnostic.

## Bringing up a new target

The intended bring-up path is incremental:

1. Obtain one canonical linked unit with `-emit-unit`.
2. Compile it with a trusted reference path, normally generated C plus a tiny
   board shell.
3. Implement the smallest candidate object milestone.
4. Validate object class, machine, ABI flags, sections, exports, imports, and
   relocations before linking.
5. Link both reference and candidate artifacts.
6. Run the omnibus and read one fixed result block.
7. Advance from constant return through arithmetic, memory, calls, globals,
   aggregates, and finally a complete backend-owned image.

The omnibus exists for targets where the normal corpus makes no sense, such as
a CH32V003-class microcontroller. It goes from “one object can be linked” to
“the backend can produce its own valid working objects” with minimal target
interaction.

The canonical omnibus roots are cumulative:

- `Stage0`: constant return and result publication;
- `Stage1`: arithmetic, comparisons, branches, shifts, loads, stores;
- `Stage2`: calls, eight arguments, stack temporaries, bounded recursion;
- `Stage3`/`RunAll`: globals, pointers, structs, methods, arrays, aggregate
  return ABI.

Results use the fixed 64-byte debugger-memory ABI in
`backend/docs/omnibus-result-abi.md`. The target does not need a console,
filesystem, allocator, or test runner.

The bring-up pipeline verifies that reference and candidate builds consume the
same unit path and SHA-256. It validates artifacts before running them and
compares completed probes and signatures, not printed prose.

### C89 reference backend

The C path is a portability and bootstrap reference, not an invitation to rely
on modern C.

Machine profiles record:

- hosted versus freestanding;
- `CHAR_BIT`;
- Renvo language-`int` width;
- pointer width;
- endianness;
- ABI;
- runtime capabilities.

Automatic profiles infer exact unsigned carriers from `<limits.h>`. Explicit
profiles fix 16-, 32-, or 64-bit language integers and pointers and emit named
negative-array static assertions. An ISO C89 language `int` cannot be eight
bits; an eight-bit device normally uses an eight-bit byte and a conforming
16-bit or wider carrier.

Generated signed operations use unsigned modular bit patterns. Do not depend
on signed overflow, implementation-defined signed right shift, or
out-of-range signed conversion. Symbol mangling must respect old linkers with
six-character external-name significance.

Use:

```sh
go run ./backend/cmd/renvoc89 --help
```

### Object target versus board

Keep these separate:

- an `ObjectTarget` defines ISA, ABI, widths, byte order, object class,
  machine, and ABI flags;
- a `Board` defines flash/RAM, startup, BSS, vectors, stack/guard, heap/OOM,
  volatile memory, debugger transport, and allowed runtime imports.

The CH32V003 profile combines RV32EC/ILP32E with 16 KiB flash and 2 KiB SRAM.
The artifact gate validates final ELF placement, LMA/VMA ranges, reserved
memory, stack collisions, vectors, entry alignment, ABI flags, imports, and
flash/RAM budgets before simulation or flashing.

Useful tools are:

- `backend/cmd/renvoc89`: C89 machine/support generation;
- `backend/cmd/renvounitc`: convert direct Go inputs into a unit, or dump a
  unit's canonical source;
- `backend/cmd/renvoboard`: board artifact validation;
- `backend/cmd/renvoresult`: decode and validate an omnibus result from a
  target memory dump;
- `backend/cmd/renvoresultgen`: generate the Go and C89 result ABI layouts;
- `backend/cmd/renvoschemagen`: generate unit readers, constants, and
  documentation from the unit schema.

## Diagnostics

Renvo diagnostics are structured by phase and code, for example
`RENVO-LOAD-...`, `RENVO-CHECK-...`, and `RENVO-BACKEND-...`.

A good diagnostic answers:

1. which file and source span;
2. which phase rejected it;
3. a stable code;
4. what is wrong;
5. whether the feature is unsupported or the program is invalid.

Preserve the earliest semantic error. Do not replace a useful loader/checker
diagnostic with a generic backend failure while crossing an arena reset or
process boundary. The self-hosted driver keeps a bounded persistent diagnostic
buffer specifically so transient compiler memory can be reclaimed without
losing the message.

Negative tests should pin the structured diagnostic. Human wording can improve,
but changing a phase or code should be deliberate.

## Documentation discipline

The useful hierarchy is:

1. current tests and source;
2. `AGENTS.md` restrictions;
3. machine-readable schemas and target contracts;
4. focused docs under `backend/docs/`;
5. overview prose.

Generated documents must be regenerated from their schema rather than edited
by hand. Architecture prose can lag implementation—for example, executable
layout changed during PIE work—so pair documentation updates with the test
that proves the property.

When a significant external probe finds a bug:

- preserve the minimal reproducer;
- open one actionable issue per independent cause;
- clearly label deliberately unsupported features;
- turn fixes into focused PRs;
- close issues from the PR body;
- retain the probe as a regression.

## Before opening or merging a PR

Check:

- The change is based on current `main`.
- Unrelated worktree and `sandbox/` files are untouched.
- Every miscompile has a minimal regression.
- Unsupported features fail with a clear diagnostic.
- Host and self-hosted behavior agree.
- Canonical units remain deterministic.
- The direct backend subset has not accidentally grown without documentation.
- Relevant package and corpus tests pass.
- Relevant resource and performance gates pass.
- Windows/Darwin changes have native coverage.
- Standalone/bundled changes work away from the repository.
- Release help remains useful with no arguments.
- No compiler output is hardcoded or copied from the compiler itself.
- The PR explains architecture impact, user impact, and validation.

Main is protected by a merge queue. A green PR may still need to be marked
ready and enqueued; wait for the queue validation and confirm the merge commit
rather than treating “queued” as “merged.”

The Actions workflow handles `pull_request`, `merge_group`, and pushes to
`main`. Pull requests run the fast `preflight` and all nine compiler self-hosting performance gates in
separate Linux, Windows, macOS, and virtual-target workflows concurrently and can enter the merge queue once both
pass. The exact prospective `main` commit then runs all nine Tier 1 performance targets and every platform, package, and
frontend job before the final `Required` job succeeds. The frontend corpus
starts as soon as preflight passes because it has no data dependency on the
backend or macOS package jobs. This keeps full testing as a pre-merge gate
without running the same suite twice or serializing independent jobs.
Repository rules should require the single stable `Required` context with
strict/merge-queue validation; do not require individual job names as they are
implementation details. Keep the post-merge `main` run enabled as a backstop,
not as the first place a change is validated. New commits cancel stale
in-progress PR runs, while merge-queue and `main` runs are never cancelled.

Windows CI executes amd64 and 386 programs natively and constructs and validates
ARM64 PE images. Full Windows/ARM64 execution requires a native ARM64 Windows
runner; image validation must not be described as runtime coverage.

## Final perspective

Renvo's unusual value comes from keeping several promises at once:

- familiar Go-like source;
- a compiler small enough to self-host cheaply;
- one-file offline toolchains;
- direct cross-target output;
- deterministic frontend/backend boundaries;
- realistic paths to old C compilers and tiny boards;
- regressions that test semantics rather than artifacts.

Most architectural mistakes make one of those promises easier by quietly
abandoning another. The best Renvo changes solve the immediate bug while
preserving all of them.
