package main

func renvo0712Fact(n int) int {
	if n <= 1 {
		return 1
	}
	return n * renvo0712Fact(n-1)
}

func appMain(args []string) int {
	if renvo0712Fact(5) != 120 {
		print("RENVO-0712 recursive diagnostic failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
