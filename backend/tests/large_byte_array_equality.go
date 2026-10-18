package main

func appMain(args []string) int {
	var x, y [65536]byte
	for j := 0; j < len(x); j++ {
		if x[j] != 0 || y[j] != 0 {
			print("FAIL initial zero\n")
			return 1
		}
	}
	if x != y {
		print("FAIL zero equality\n")
		return 1
	}
	x[65535], y[65535] = 7, 7
	for j := 0; j < len(x); j++ {
		if x[j] != y[j] {
			print("FAIL scalar equality\n")
			return 1
		}
	}
	if x != y {
		print("FAIL typed equality\n")
		return 1
	}
	y[65535] = 8
	if x == y {
		print("FAIL typed difference\n")
		return 1
	}
	print("PASS\n")
	return 0
}
