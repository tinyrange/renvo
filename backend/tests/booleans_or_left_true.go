package main

var bool12Touched int

func bool12Probe() bool {
	bool12Touched += 1
	return false
}

func appMain(args []string) int {
	if true || bool12Probe() {
		if bool12Touched != 0 {
			print("booleans_12 side\n")
			return 1
		}
		print("PASS\n")
		return 0
	}
	print("booleans_12 branch\n")
	return 2
}
