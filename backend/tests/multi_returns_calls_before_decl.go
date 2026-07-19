package main

func appMain(args []string) int {
	left, right := renvo1009Later(6)
	if left != 7 || right != 12 {
		print("RENVO-1009 call before decl returns failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func renvo1009Later(n int) (int, int) {
	return n + 1, n * 2
}
