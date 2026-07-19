package main

var bool15Count int

func bool15Touch() bool { bool15Count += 1; return false }
func appMain(args []string) int {
	if true || bool15Touch() {
		if bool15Count != 0 {
			print("booleans_15 side\n")
			return 1
		}
		print("PASS\n")
		return 0
	}
	print("booleans_15 branch\n")
	return 2
}
