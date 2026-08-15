# C11 frontend

This package is an alternate source frontend, not a second compiler pipeline.
It adapts C declarations and statements to the same checked package syntax used
by the Go frontend. Package checking, symbol resolution, whole-program linking,
unit serialization, dead-code elimination, and backend code generation remain
shared. Consequently, C and Go files in one directory can call each other's
package-level functions directly.

The adapter is intentionally compact:

- tokens are three integers containing a packed kind/line and two byte offsets;
- token text is retained only in the original source buffer;
- translation is direct and does not build a pointer-heavy AST;
- there is no host C compiler, linker, or target C runtime dependency;
- generated source is released with the package's existing transient arena.

The initial executable subset includes scalar and basic pointer declarations,
fixed arrays, direct functions and calls, local/global variables, integer
expressions, `if`, `while`, expression-form `for`, and returns.
Header directive lines and function prototypes are accepted so declarations can
be supplied by Go files in the same package. Macro expansion, aggregate layout,
full C integer conversions, variadics, function pointers, C string/pointer
semantics, and libc compatibility are future work. Unsupported constructs must
fail in this frontend rather than reaching a backend.
