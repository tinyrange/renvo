package main

func appMain(args []string) int {
	value := 0x30
	if value != 48 {
		print("integer_literals_08 value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
