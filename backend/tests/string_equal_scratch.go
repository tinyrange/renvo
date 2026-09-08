package main

func equalStrings(a, b string) bool { return a == b }

func appMain() int {
	a := "native backend"
	b := "other backend"
	if !equalStrings(a[7:], b[6:]) || equalStrings(a, b) {
		print("FAIL string equality scratch\n")
		return 1
	}
	print("PASS\n")
	return 0
}
