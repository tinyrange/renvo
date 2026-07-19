package main

func appMain(args []string) int {
	x := 3
	y := 3
	p := &x
	q := &y
	if !(p != q) {
		print("RENVO-0198 pointer_inequality_with_different_addresses failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
