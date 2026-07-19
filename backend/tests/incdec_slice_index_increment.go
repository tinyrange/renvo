package main

func appMain(args []string) int {
	xs := []int{2, 4, 6}
	index := 1
	xs[index]++
	if xs[0] != 2 || xs[1] != 5 || xs[2] != 6 {
		print("RENVO-INCDEC-010 slice index increment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
