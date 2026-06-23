package main

func nestedArgValue(v int) int {
	if v == 7 {
		return 3
	}
	return 4
}

func nestedArgCombine(a int, b int) int {
	if a == 11 {
		if b == 3 {
			return 1
		}
	}
	return 0
}

func nestedArgUse(a int, b int) int {
	return nestedArgCombine(a, nestedArgValue(b))
}

func appMain() int {
	if nestedArgUse(11, 7) == 1 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
