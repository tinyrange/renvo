The fixture uses Go 1.25.5's `src/slices/slices.go`, `sort.go`,
`zsortordered.go`, `zsortanyfunc.go`, and `iter.go`, plus `src/maps/iter.go`. Only the `cmp`, `iter`, and `math/bits` import
paths are changed to fixture-local dependencies. The `cmp` source is unchanged.
`bits/mul.go` contains the upstream Mul, Mul32 and Mul64 implementations and
their UintSize constant; `bits/len.go` contains the upstream Len family and
its lookup table. These dependencies keep the fixture independent of gaps
in Renvo's math/bits. The Go Authors' BSD license is included in LICENSE.

This tests upstream generic algorithms independently of a standard-library
runtime overlay, including overlapping edits, callbacks, dependent inference,
unsafe pointer conversions, named slices, and several element types. Sorting
covers ordered and callback implementations, stable ordering of records,
NaNs, and binary searches with different element and target types.

Iterator coverage uses the upstream All, Backward, Values, AppendSeq, Collect,
Sorted, SortedFunc, SortedStableFunc and Chunk implementations, and map
All/Keys/Values/Insert/Collect. The local iter package contains the upstream
public Seq/Seq2 type declarations; it does not include the coroutine-dependent
Pull implementations. Runtime overlay integration remains separate work.
