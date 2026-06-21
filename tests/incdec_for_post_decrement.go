package main

func appMain(args []string) int {
	sum := 0
	for i := 4; i > 0; i-- {
		sum = sum + i
	}
	if sum != 10 {
		print("RTG-INCDEC-004 for post decrement failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
