package main

func i8086CallWords(a string, av int, b string, bv int, c string, cv int) int {
	if a != "a" || av != 1 || b != "b" || bv != 2 || c != "c" || cv != 3 {
		return 1
	}
	return 0
}

func appMain() int {
	if i8086CallWords("a", 1, "b", 2, "c", 3) != 0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
