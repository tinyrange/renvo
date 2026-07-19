package main

func appMain(args []string) int {
	value := 3 << 4
	if value != 48 {
		print("integer_literals_18 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
