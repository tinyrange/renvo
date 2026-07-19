package main

func renvo1003Pick(n int) (int, bool) {
	if n > 4 {
		return n + 1, true
	}
	return n - 1, false
}

func appMain(args []string) int {
	value, ok := renvo1003Pick(5)
	if !ok || value != 6 {
		print("RENVO-1003 int bool branch failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
