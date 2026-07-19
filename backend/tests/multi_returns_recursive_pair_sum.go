package main

func renvo1008Sum(n int) (int, int) {
	if n == 0 {
		return 0, 0
	}
	sum, count := renvo1008Sum(n - 1)
	return sum + n, count + 1
}

func appMain(args []string) int {
	sum, count := renvo1008Sum(4)
	if sum != 10 || count != 4 {
		print("RENVO-1008 recursive pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
