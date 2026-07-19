package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 5; i++ {
		sum = sum + i
	}
	if sum != 10 {
		print("RENVO-INCDEC-003 for post increment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
