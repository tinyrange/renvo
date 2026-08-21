# Compiler bugs

## errors.As cannot assign general typed targets

The Renvo errors.As implementation can traverse wrappers and invoke custom As methods, but cannot implement Go's required assignment into arbitrary pointers such as `**MyError`. The current reflect surface has no assignability test or Value.Set/interface assignment intrinsic. This is compiler/runtime reflection work rather than an errors package semantic exception.
## Renvo cannot construct or inspect IEEE-754 special values for strconv

`strconv.ParseFloat("NaN", 64)` attempts to construct NaN from a runtime zero divided by zero, but the Renvo-compiled regression produces a value equal to itself. The standard library also lacks float bit-cast intrinsics needed to construct NaN/Inf payloads and implement exact shortest-round-trip formatting. Hex-float parsing works. Add target-correct Float32bits/Float64bits and inverse intrinsics (or equivalent compiler support), then finish special values and a Ryu/round-trip formatter.
## Float literals lose precision outside constant contexts

Frontend. A float literal used in a runtime (non-constant-folded) context
appears to be truncated to roughly 7 significant digits instead of keeping
full float64 precision. Constant-expression comparisons keep full precision.

Reproduced with the Renvo bootstrap on darwin/arm64:

```go
package main

import "math"

func main() {
	s := math.Sqrt(2.0) // 1.4142135623730951
	if s < 1.4142125623730951 {
		print("s < lower\n") // not printed (correct)
	}
	if s > 1.4142145623730951 {
		print("s > upper\n") // printed (wrong): the literal behaves like ~1.414214
	}
	l := 1.41421356237309
	r := 1.414213562373096
	if l == r {
		print("equal\n") // printed (wrong): literals differing at 1e-13 compare equal
	}
}
```

Observed while validating std/math; see also the FormatFloat and NaN
reports below, which were found in the same investigation.

## Non-strict float comparisons materialized as bool values are wrong for equal operands

Backend. When a float64 comparison `a <= b` or `a >= b` is used as a
boolean value (function return or assignment) and the operands are equal,
the result is false. The same comparison in an if condition branches
correctly, and strict `<`, `>`, `!=` materialize correctly.

Minimal reproducer (bootstrap, darwin/arm64):

```go
package main

func le(a, b float64) bool { return a <= b }

func main() {
	if le(3, 3) {
		print("true\n")
	} else {
		print("false\n") // printed; want true
	}
}
```

`return a-b <= c` inside a three-float-parameter function shows the same
failure, while `if a-b <= c { return true }; return false` in the identical
function returns the correct answer.

## No IEEE NaN can be produced by arithmetic on native targets

Hosted runtime / backend. On compiled native targets, 0.0/0.0 yields +Inf
instead of NaN, and Inf-Inf does not produce NaN either, so IsNaN(x) (x != x)
can never be true. Host Go produces NaN for both. This blocks math.NaN(),
math.Sqrt(-1), math.Log(-1), and math.Mod(x, 0) semantics in compiled
programs.

```go
package main

import "fmt"

func main() {
	var z float64
	fmt.Println(z/z != z/z) // host Go: true; compiled Renvo: false
}
```

Found while implementing std/math; the package currently cannot be claimed
NaN-complete in compiled form.

## strconv.FormatFloat emits wrong digits under Renvo

Compiled output of strconv.FormatFloat produces incorrect digit strings,
both in 'g' and 'f' modes, while simple values format correctly.

Observed under the bootstrap on darwin/arm64:

- FormatFloat(math.Sqrt(2), 'g', 17, 64) printed "2"
- FormatFloat(1024, 'g', 17, 64) printed "10"
- FormatFloat(1.0/3.0, 'f', 6, 64) printed "0.250000" (want 0.333333)
- FormatFloat(3.141592653589793, 'f', 4, 64) printed "3.0000" (want 3.1416)

The underlying computation was verified correct by comparing against
expected constants with tolerance checks, so this is a formatting bug, not
arithmetic. It may share a root cause with the float literal precision
report above.

