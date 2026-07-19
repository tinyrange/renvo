package main

const intLitLoopLimit = 4

func appMain(args []string) int {
	sum := 0
	for i := 0; i < intLitLoopLimit; i = i + 1 {
		sum += i
	}
	if sum != 6 {
		print("integer_literals_16 loop\n")
		return 1
	}
	print("PASS\n")
	return 0
}
