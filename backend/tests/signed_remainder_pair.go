package main

func check(a, b int) bool {
	q, r := a/b, a%b
	return q*b+r == a
}

func appMain() int {
	if !check(-37, -11) || !check(-37, 11) || !check(37, -11) || !check(37, 11) {
		print("FAIL signed remainder pair\n")
		return 1
	}
	print("PASS\n")
	return 0
}
