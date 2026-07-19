package main

func appMain(args []string) int {
	value := 3 + 4 + 5
	if value != 12 {
		print("integer_literals_09 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
