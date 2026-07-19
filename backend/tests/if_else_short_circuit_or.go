package main

var renvo0364Touched int

func renvo0364Mutate() bool { renvo0364Touched = 1; return false }
func appMain(args []string) int {
	ok := false
	if true || renvo0364Mutate() {
		ok = true
	}
	if !ok || renvo0364Touched != 0 {
		print("RENVO-0364 short or failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
