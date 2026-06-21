package main

func rtg1050Walk(n int) (int, int) {
	if n == 0 {
		return 0, 1
	}
	sum, scale := rtg1050Walk(n - 1)
	return sum + n, scale * 2
}

func rtg1050Use(sum int, scale int) int {
	return sum + scale
}

func appMain(args []string) int {
	if rtg1050Use(rtg1050Walk(3)) != 14 {
		print("RTG-1050 recursive tuple pipeline failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
