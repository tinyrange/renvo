package main

func renvo0529Value(a int, b int) int {
	return a*b + a - b
}

func appMain(args []string) int {
	if renvo0529Value(6, 4) != 26 {
		print("RENVO-0529 expression return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
