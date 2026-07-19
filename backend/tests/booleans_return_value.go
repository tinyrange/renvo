package main

func bool06() bool { return true }
func appMain(args []string) int {
	if !bool06() {
		print("booleans_06 return\n")
		return 1
	}
	print("PASS\n")
	return 0
}
