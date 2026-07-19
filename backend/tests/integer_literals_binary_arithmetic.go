package main

func appMain(args []string) int {
	value := 0b1000 + 0b0011
	if value != 11 {
		print("integer_literals_07 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
