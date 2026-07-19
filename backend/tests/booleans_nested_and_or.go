package main

func appMain(args []string) int {
	if false || (true && 5 == 5) {
		print("PASS\n")
		return 0
	}
	print("booleans_16 nested\n")
	return 1
}
