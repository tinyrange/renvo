package main

func appMain(args []string) int {
	sum := 0
	for i := 4; i > 0; i = i - 1 {
		sum = sum + i
	}
	if sum != 10 {
		print("RENVO-0402 downward for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
