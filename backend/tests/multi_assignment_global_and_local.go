package main

var renvo1030Global int

func renvo1030Values() (int, int) {
	return 8, 15
}

func appMain(args []string) int {
	local := 0
	renvo1030Global, local = renvo1030Values()
	if renvo1030Global != 8 || local != 15 {
		print("RENVO-1030 global local assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
