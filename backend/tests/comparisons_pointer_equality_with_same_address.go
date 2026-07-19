package main

func appMain(args []string) int {
	x := 3
	p := &x
	q := &x
	if !(p == q) {
		print("RENVO-0197 pointer_equality_with_same_address failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
