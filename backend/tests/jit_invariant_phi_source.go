package main

// Readonly invariant values can feed a distinct loop-carried word. Preserve
// the source value independently of its slot number and intermediate loaded data.
func appMain(args []string) int {
	values := make([]int, 4)
	values[1], values[2] = 9, 19
	sources := make([]int, 4)
	sources[3] = 2
	address, total := 1, 0
	for i := 0; i < 3; i++ {
		loaded := values[address]
		address = loaded
		total += loaded
		address = sources[3]
	}
	if address != 2 || sources[3] != 2 || total != 47 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
