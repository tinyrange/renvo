package main

func appMain(args []string) int {
	a := 3
	b := 8
	a, b = b, a
	if a != 8 || b != 3 {
		print("RTG-1021 swap locals failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
