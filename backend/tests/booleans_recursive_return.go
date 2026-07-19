package main

func bool25(n int) bool {
	if n == 0 {
		return true
	}
	return bool25(n - 1)
}
func appMain(args []string) int {
	if !bool25(4) {
		print("booleans_25 recursion\n")
		return 1
	}
	print("PASS\n")
	return 0
}
