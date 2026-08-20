# Compiler bugs

## Backend cannot emit unary bitwise complement in SHA-256 expression

The Renvo-compiled `crypto/sha256` regression failed with RENVO-BACKEND-003 while emitting `choice := (e & f) ^ (^e & g)`. Rewriting `^e` as `e ^ uint32(0xffffffff)` compiled successfully. The backend should support Go unary bitwise complement for integer operands.

## Backend cannot pass imported function as bufio.Scanner split callback

The `std_bufio_scanner` frontend regression failed with RENVO-BACKEND-003 while emitting `words.Split(bufio.ScanWords)` (lowered display: `words.Split(ScanWords)`). `Scanner.Split` accepts a named function type, so imported package functions must be usable as function values and call arguments.

## Backend cannot initialize Scanner function-valued field

After removing the direct `Scanner.Split` call, the `std_bufio_scanner` regression still failed with RENVO-BACKEND-003 while emitting `return &Scanner{r: r, split: ScanLines, maxTokenSize: MaxScanTokenSize}`. This blocks the standard `bufio.Scanner` design, which stores a `SplitFunc` callback.

## Renvo miscompiles repeated bufio.Reader.ReadLine results

Host-Go tests return `("beta", false, nil)` on the second `ReadLine` after reading `pha` from `alpha\r\nbeta\ngamma`. The Renvo-compiled regression returns the correct `beta` bytes but reports `isPrefix == true`, even with a 32-byte buffer large enough for the full input. This appears to corrupt or misassign the second result of a repeated three-result method call.

## Backend cannot assign computed bytes into a local array

The compiled `bufio.Writer.WriteRune` implementation failed with RENVO-BACKEND-003 on assignments such as `data[0] = byte(0xc0 + value/64)` when `data` was `[utf8.UTFMax]byte`. Changing only the temporary from an array to `make([]byte, utf8.UTFMax)` made the same indexed assignments compile and the regression pass.

## Closure stored in FlagSet.Usage makes flag package backend compilation fail

After adding Go-style default usage initialization, `NewFlagSet` assigned `f.Usage = func() { f.PrintDefaults() }`. The previously passing `std_flag` regression then failed with RENVO-BACKEND-003 and no statement diagnostic. Removing only that closure assignment restored compilation. Function-valued fields with closures remain a blocker for fully compatible `flag.FlagSet.Usage` defaults.

