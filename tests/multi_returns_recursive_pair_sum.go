package main

func rtg1008Sum(n int) (int, int) {
	if n == 0 {
		return 0, 0
	}
	sum, count := rtg1008Sum(n - 1)
	return sum + n, count + 1
}

func appMain(args []string) int {
	sum, count := rtg1008Sum(4)
	if sum != 10 || count != 4 {
		print("RTG-1008 recursive pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
