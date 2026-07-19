package main

var bool11Touched int

func bool11Probe() bool {
	bool11Touched += 1
	return true
}

func appMain(args []string) int {
	if false && bool11Probe() {
		print("booleans_11 branch\n")
		return 1
	}
	if bool11Touched != 0 {
		print("booleans_11 side\n")
		return 2
	}
	print("PASS\n")
	return 0
}
