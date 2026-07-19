package main

func appMain(args []string) int {
	if 0b1010 == 0x0b {
		print("integer_literals_24 neq\n")
		return 1
	}
	print("PASS\n")
	return 0
}
