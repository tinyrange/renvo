# C11 frontend

This package is an alternate source frontend, not a second compiler pipeline.
It adapts C declarations and statements to the same checked package syntax used
by the Go frontend. Package checking, symbol resolution, whole-program linking,
unit serialization, dead-code elimination, and backend code generation remain
shared. Consequently, C and Go files in one directory can call each other's
package-level functions directly.

The adapter is intentionally compact:

- tokens are three integers containing a packed kind/line and two byte offsets;
- preprocessing tokens are three integers containing spelling, source-table,
  whitespace, and hide-set IDs; macro names are interned once in an open-addressed table;
- token text is retained only in the original source buffer;
- translation is direct and does not build a pointer-heavy AST;
- there is no host C compiler, linker, or target C runtime dependency;
- generated source is released with the package's existing transient arena.

The initial executable subset includes scalar and basic pointer declarations,
fixed arrays, direct functions and calls, local/global variables, integer
expressions, `if`, `while`, expression-form `for`, and returns.
Header directive lines and function prototypes are accepted so declarations can
be supplied by Go files in the same package. The output-only preprocessing path
implements translation-phase splicing/comments, recursive includes and
`include_next`, guards/`pragma once`, object/function/variadic macros, rescanning
with hide sets, stringizing/pasting, conditionals and integer expressions,
`#line`, target predefines, and transitive dependency capture. Object translation
continues to use its bounded declaration adapter until the M3 C type system can
consume the complete preprocessed header stream. Aggregate layout, full C
integer conversions, function pointers, C string/pointer semantics, and libc
compatibility remain future work. Unsupported constructs must fail in this
frontend rather than reaching a backend.
