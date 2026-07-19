package main

func manyWordArgs(a int, b int, c int, d int, e int, f int, g int, h int, i int) int {
	return a + b + c + d + e + f + g + h + i
}

func appMain() int {
	sum := 0
	for i := 0; i < 90000; i++ {
		sum = manyWordArgs(1, 2, 3, 4, 5, 6, 7, 8, 9)
	}
	if sum != 45 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
