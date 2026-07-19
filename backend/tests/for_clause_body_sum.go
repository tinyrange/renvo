package main

func appMain(args []string) int {
	sum := 0
	for i := 1; i <= 4; i = i + 1 {
		sum += i
	}
	if sum != 10 {
		print("RENVO-0411 body sum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
