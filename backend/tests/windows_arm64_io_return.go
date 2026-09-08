package main

func appMain() int {
	// Both writes must return to their caller, including the reused I/O helper.
	if write(1, []byte("PA"), -1) != 2 {
		return 1
	}
	print("SS\n")
	return 0
}
