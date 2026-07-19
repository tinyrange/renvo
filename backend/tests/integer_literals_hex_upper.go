package main

func appMain(args []string) int {
	value := 0x2A + 1
	if value != 43 {
		print("integer_literals_05 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
