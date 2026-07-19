package main

func appMain(args []string) int {
	value := 0b101010
	if value != 42 {
		print("integer_literals_06 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
