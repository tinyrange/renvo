package main

func check(a, b, c, d, e, f, g, h, i, j int) bool {
	if a != 1 {
		print("FAIL a\n")
		return false
	}
	if b != 2 {
		print("FAIL b\n")
		return false
	}
	if c != 3 {
		print("FAIL c\n")
		return false
	}
	if d != 4 {
		print("FAIL d\n")
		return false
	}
	if e != 5 {
		print("FAIL e\n")
		return false
	}
	if f != 6 {
		print("FAIL f\n")
		return false
	}
	if g != 7 {
		print("FAIL g\n")
		return false
	}
	if h != 8 {
		print("FAIL h\n")
		return false
	}
	if i != 9 {
		print("FAIL i\n")
		return false
	}
	if j != 10 {
		print("FAIL j\n")
		return false
	}
	return true
}

func appMain(args []string) int {
	if !check(1, 2, 3, 4, 5, 6, 7, 8, 9, 10) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
