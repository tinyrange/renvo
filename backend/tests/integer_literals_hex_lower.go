package main

func appMain(args []string) int {
	value := 0x2a
	if value != 42 {
		print("integer_literals_04 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
