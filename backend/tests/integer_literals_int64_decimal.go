package main

func appMain(args []string) int {
	var x int64 = 12345
	if x != int64(12345) {
		print("integer_literals_21 int64\n")
		return 1
	}
	print("PASS\n")
	return 0
}
