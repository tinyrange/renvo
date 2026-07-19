package main

func appMain(args []string) int {
	value := 0 + 1
	if value != 1 {
		print("integer_literals_01 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
