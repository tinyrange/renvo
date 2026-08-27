# C11 frontend

This package is an alternate source frontend, not a second compiler pipeline.
It adapts C declarations and statements to the same checked package syntax used
by the Go frontend. Package checking, symbol resolution, whole-program linking,
unit serialization, dead-code elimination, and backend code generation remain
shared. Ordinary mixed packages follow cgo's explicit source contract. The
comment immediately preceding a standalone `import "C"` is parsed as a C
preamble, and only functions declared there are visible through that Go file's
`C.name` namespace. Go functions carrying an adjacent `//export name` directive
are declared in a synthetic `_cgo_export.h`; C files include that header to call
them. The logical namespace is separate, so identically named Go and C
declarations can coexist.

The adapter is intentionally compact:

- tokens are three integers containing a packed kind/line and two byte offsets;
- preprocessing tokens are three integers containing spelling, source-table,
  whitespace, and hide-set IDs; macro names are interned once in an open-addressed table;
- token text is retained only in the original source buffer;
- translation is direct and does not build a pointer-heavy AST;
- there is no host C compiler, linker, or target C runtime dependency;
- generated source is released with the package's existing transient arena.

The executable subset includes scalar and basic pointer declarations, fixed
arrays, direct functions and calls, local/global variables, integer expressions,
and ISO control flow including `if`/`else`, `while`, `do`/`while`, both forms of
`for`, C-fallthrough `switch`, labels/`goto`, break/continue, and returns. M3 adds
integer-ID C types in flat arenas, typedef names, named and anonymous structures,
x86_64 LP64 field/array layout, anonymous aggregate-member promotion, enum
constants, recursive parenthesized/array/function declarators, function-pointer
types, and constant folding for `sizeof`, `_Alignof`, array bounds, and
direct-field `__builtin_offsetof`. SysV union storage uses alignment-preserving
carriers with typed overlapping access, and integer/`_Bool` bitfields use
allocation-unit layout plus generated read/modify/write accessors. File-scope
prototypes and tentative definitions coalesce by C linkage rules; incomplete
tentative arrays complete at the end of the translation unit, and block
`static` objects are hoisted with scope-safe internal names.
The `cc` executable path is hermetic and accepts one or more C files without a
Go module. It preprocesses against the embedded `libc/include` tree, selects
implementation files from `libc/src` only for headers actually reached by the
translation unit, and lowers them through the same frontend. Target descriptors
select ILP32, LP64, or Windows LLP64 widths and define the conventional
architecture/operating-system macros used by guarded library implementations.
The C ordinary-identifier namespace is kept separate from Go's predeclared
identifiers, while compiler-generated library bridges retain access to the
portable read, write, and exit runtime operations.
An active `#pragma go "filename.go"` adds that same-directory Go source and its
transitive imports to an executable C build. Quoted, angle-bracket, and bare
filenames are accepted; inactive conditional branches do not add sources, and
adjacent Go files remain excluded unless explicitly named. This C-first mode
intentionally retains direct shared-name linkage for its selected Go adapters.

The browser language service indexes original C and header byte spans rather
than the generated Go-shaped source. It provides live preprocessing/scanning
diagnostics, scoped and aggregate-member completion, inferred-type and
documentation hover, signature help, definitions, references, and rename.
External declarations, project headers, macros, tags, enumerators, local
shadowing, and translation-unit-local `static` names retain distinct bindings.
Mixed projects also navigate across an `extern` declaration and a matching Go
function selected with `#pragma go`. Preprocessing can optionally retain a
token-level origin map so diagnostics produced after macro/include expansion
are reported against their original C or header spelling; ordinary compiler
builds do not pay for that map.

Aggregate initialization supports positional and chained designated
structure/array members, repeated-designator overwrite, inferred array bounds,
nested braces, bitfields, union active members, and structure/union compound
literals.
Qualifier IDs are retained on scalar, aggregate, and individual pointer levels;
`const` lvalue mutation and invalid non-pointer `restrict` are diagnosed, while
volatile loads/stores remain explicit in the shared unoptimized object path.
The scalar expression path applies integer promotions, signed/unsigned rank
selection, assignment/return/call conversions, `_Bool` normalization, and
conditional-expression result typing. Adjacent ordinary strings are joined and
octal/hex escapes are decoded before their single trailing NUL is emitted.
Linux/x86_64 thread-duration definitions use libc thread-specific keys behind a
process-wide initialization mutex; linked pthread regressions verify independent
state and persistence under host and stage3 compilers.
Header directive lines and function prototypes are accepted. In an ordinary
mixed package a prototype can be supplied by an explicitly exported Go
function; `#pragma go` projects retain their direct C-first binding. The
output-only preprocessing path
implements translation-phase splicing/comments, recursive includes and
`include_next`, guards/`pragma once`, object/function/variadic macros, rescanning
with hide sets, stringizing/pasting, conditionals and integer expressions,
`#line`, target predefines, and transitive dependency capture. Object translation
consumes that stream directly: declarations from quoted project headers are
retained, while large system headers remain bounded by demand-selecting the
referenced external prototypes. The frozen M3 foundation is complete: it also
covers block namespaces, character-array strings, function pointers, expression
sequencing, promotions and conversions, same-unit external thread declarations,
atomic/remaining qualifier validation, and variadic definitions whose optional
pack is not consumed. Direct external variadic calls are arity-specialized after
C default argument promotion; the shared x86_64 object-call descriptor classifies
integer and floating arguments, sets the SysV vector count, and aligns every
foreign call boundary. Executable variadic definitions use a target-width word
pack supporting promoted pointers, integers, and floating-point values on both
32- and 64-bit targets. Cross-object thread or static string addresses need
later relocations.
Attributes and the other GNU C surface remain M4 work.
Unsupported constructs must fail in this frontend rather than reaching a
backend.
