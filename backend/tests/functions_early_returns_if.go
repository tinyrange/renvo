package main

func renvo0496Check(x int) int {
	if x > 3 {
		return 21
	}
	return 0
}
func appMain(args []string) int {
	if renvo0496Check(4) != 21 {
		print("RENVO-0496 early return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
