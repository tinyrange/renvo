package main

var bool14Count int

func bool14Touch() bool { bool14Count += 1; return true }
func appMain(args []string) int {
	if false && bool14Touch() {
		print("booleans_14 branch\n")
		return 1
	}
	if bool14Count != 0 {
		print("booleans_14 side\n")
		return 2
	}
	print("PASS\n")
	return 0
}
