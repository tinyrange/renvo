package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 4; i = i + 1 {
		sum += i * 3
	}
	if sum != 18 {
		print("arithmetic_15 loop\n")
		return 1
	}
	print("PASS\n")
	return 0
}
