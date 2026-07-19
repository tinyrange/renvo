# C89 machine-profile generator

`renvoc89` exposes the C backend's machine-contract prefix as a deterministic
command-line tool. Automatic mode selects exact unsigned carrier types from
`<limits.h>`. Explicit mode fixes the RENVO language-int and data-pointer widths
and emits named negative-array assertions, so an incompatible compiler fails
while translating the generated C rather than silently changing semantics.

Generate a freestanding profile that follows the C implementation's native
`unsigned int` and pointer widths:

```sh
go run ./backend/cmd/renvoc89 -mode automatic -name bootstrap \
  -abi vendor-c -endian little -o renvo_support.c
```

Generate a reproducible 32-bit profile for an embedded toolchain:

```sh
go run ./backend/cmd/renvoc89 -mode explicit -name rv32ec -abi ilp32e \
  -int 32 -pointer 32 -endian little -runtime result -o renvo_support.c
```

The output defaults to a complete support unit containing the defined signed
arithmetic helpers. `-preamble-only` emits only carrier selection, named static
checks, endianness, hosted/freestanding state, ABI identity, and runtime
capabilities. Running the command without arguments prints all options.

ISO C89 cannot provide an 8-bit language `int`; explicit profiles therefore
accept 16, 32, or 64 bits. An 8-bit MCU profile describes its byte/device class
while selecting a conforming wider C carrier for the language integer model.
