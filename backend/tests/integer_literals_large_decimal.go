package main

func appMain(args []string) int {
	value := 1234567 - 7
	if value != 1234560 {
		print("integer_literals_02 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
