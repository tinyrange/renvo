package main

// The closed-program interface comparison includes this comparable type.
// Its generated comparison body exceeds the AArch64 B.cond range.
type WideComparison [12000]int

func equalMixed(value int, other any) bool { return value == other }
func appMain(args []string) int {
	if !equalMixed(42, any(42)) || equalMixed(42, any("42")) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
