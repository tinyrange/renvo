package main

func appMain(args []string) int {
	value := 17 % 5
	if value != 2 {
		print("integer_literals_13 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
