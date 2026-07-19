package main

func appMain(args []string) int {
	value := -19 + 4
	if value != -15 {
		print("integer_literals_03 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
