# C89 machine profiles

The C bootstrap backend uses an explicit machine contract rather than names
such as `c32`. A profile records hosted versus freestanding execution,
`CHAR_BIT`, RTG language-int width, pointer width, endianness, ABI, and the
runtime operations supplied by target composition.

`target.C89AutomaticProfile` selects exact unsigned carrier types from
`<limits.h>` while inheriting the C implementation's `int` and pointer widths.
It is convenient for initial hosted exploration, but the selected compiler is
part of the resulting target contract.

`target.C89ExplicitProfile` fixes language-int and pointer widths to 16, 32, or
64 bits. Its generated C89 preamble uses named negative-array assertions, so a
mismatched compiler fails translation with an assumption such as
`rtg_assumption_pointer_width`. An ISO C profile cannot request an 8-bit `int`;
an 8-bit device class normally means eight-bit bytes with a conforming 16-bit
or wider language `int`.

Both forms require descriptor-supplied byte order and ABI. Generated output
uses unsigned carrier types and does not infer signed representation, char
signedness, or object ABI. The preamble is freestanding-safe, deterministic,
strict C89, and requires no libc symbols. This establishes the machine-profile
and compile-time-assumption layer of #19; unit-to-C operation emission remains
the next layer.
