package main

func renvo1006Select(flag bool) (int, int) {
	if flag {
		return 8, 1
	}
	return 3, 5
}

func appMain(args []string) int {
	a, b := renvo1006Select(false)
	c, d := renvo1006Select(true)
	if a+b != 8 || c-d != 7 {
		print("RENVO-1006 early tuple paths failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
