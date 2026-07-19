package main

func appMain(args []string) int {
	value := (2 + 3) * (4 + 1)
	if value != 25 {
		print("integer_literals_14 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
