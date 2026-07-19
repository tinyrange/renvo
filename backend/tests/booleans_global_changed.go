package main

var bool21Global bool

func appMain(args []string) int {
	bool21Global = true
	if !bool21Global {
		print("booleans_21 global\n")
		return 1
	}
	print("PASS\n")
	return 0
}
