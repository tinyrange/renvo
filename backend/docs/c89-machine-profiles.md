# C89 machine profiles

The C bootstrap backend uses an explicit machine contract rather than names
such as `c32`. A profile records hosted versus freestanding execution,
`CHAR_BIT`, RENVO language-int width, pointer width, endianness, ABI, and the
runtime operations supplied by target composition.

`target.C89AutomaticProfile` selects exact unsigned carrier types from
`<limits.h>` while inheriting the C implementation's `int` and pointer widths.
It is convenient for initial hosted exploration, but the selected compiler is
part of the resulting target contract.

`target.C89ExplicitProfile` fixes language-int and pointer widths to 16, 32, or
64 bits. Its generated C89 preamble uses named negative-array assertions, so a
mismatched compiler fails translation with an assumption such as
`renvo_assumption_pointer_width`. An ISO C profile cannot request an 8-bit `int`;
an 8-bit device class normally means eight-bit bytes with a conforming 16-bit
or wider language `int`.

Both forms require descriptor-supplied byte order and ABI. Generated output
uses unsigned carrier types and does not infer signed representation, char
signedness, or object ABI. The preamble is freestanding-safe, deterministic,
strict C89, and requires no libc symbols.

`RenderC89Support` extends the preamble with the signed-integer operations
needed by an emitter. RENVO signed values remain modular bit patterns in an
unsigned carrier. Comparisons, division, remainder, sign extension, and
arithmetic right shift are implemented without C signed overflow, signed
right shift, out-of-range signed conversion, or shifts by the carrier width.
The target runtime may override `RENVO_C_DIVZERO()` to install its trap path.
Readable helper names are macros over external symbols of five characters or
fewer so linkers with six-character external-name significance cannot merge
distinct helpers.

`target.MangleC89Symbols` applies the same constraint to linked-unit symbols.
It sorts canonical package-qualified names and assigns unique monocase names
from the six-character `rg0000` namespace. Generated declarations therefore
do not inherit long package paths or punctuation, and input traversal order
cannot change the mapping. The emitter retains the canonical-to-C table for
exports and diagnostics; a documented toolchain profile may select a wider
significant-name limit.

This establishes the machine-profile, compile-time-assumption, and defined
integer-semantics and symbol-policy layers of #19; unit-to-C operation emission
remains the next layer.
