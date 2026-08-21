# Compiler bugs

## errors.As cannot assign general typed targets

The Renvo errors.As implementation can traverse wrappers and invoke custom As methods, but cannot implement Go's required assignment into arbitrary pointers such as `**MyError`. The current reflect surface has no assignability test or Value.Set/interface assignment intrinsic. This is compiler/runtime reflection work rather than an errors package semantic exception.
## Renvo cannot construct or inspect IEEE-754 special values for strconv

`strconv.ParseFloat("NaN", 64)` attempts to construct NaN from a runtime zero divided by zero, but the Renvo-compiled regression produces a value equal to itself. The standard library also lacks float bit-cast intrinsics needed to construct NaN/Inf payloads and implement exact shortest-round-trip formatting. Hex-float parsing works. Add target-correct Float32bits/Float64bits and inverse intrinsics (or equivalent compiler support), then finish special values and a Ryu/round-trip formatter.

