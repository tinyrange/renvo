package main

func appMain(args []string) int {
	sum := 0
	for i := 5; i > 0; i -= 2 {
		sum = sum + i
	}
	if sum != 9 {
		print("RENVO-0406 post subtraction failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
