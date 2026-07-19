package main

func appMain(args []string) int {
	if !(1.5 < 2.5) {
		print("float_literals_08 less\n")
		return 1
	}
	print("PASS\n")
	return 0
}
