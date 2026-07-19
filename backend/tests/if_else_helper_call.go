package main

func renvo0362Ok() bool { return 6*7 == 42 }
func appMain(args []string) int {
	x := 0
	if renvo0362Ok() {
		x = 12
	}
	if x != 12 {
		print("RENVO-0362 helper condition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
