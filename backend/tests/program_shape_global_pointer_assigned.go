package main

var shape17Ptr *int

func appMain(args []string) int {
	x := 31
	shape17Ptr = &x
	if *shape17Ptr != 31 {
		print("program_shape_17 ptr\n")
		return 1
	}
	print("PASS\n")
	return 0
}
