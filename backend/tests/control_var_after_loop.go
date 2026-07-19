package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 4; i++ {
		sum += i
	}
	var after int
	after = 17
	if sum != 6 || after != 17 {
		print("control var after loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
