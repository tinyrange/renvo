package main

// Supplemental parsed-source test for resident-slice accounting. The native
// frame/foreign-call regression is TestNativeResidentTimerMixedLoops, including
// shared write epoch pointers and a precise fault after nonzero progress.
func appMain(args []string) int {
	initial, global, slice := uint64(1001), uint64(1001), uint64(0)
	sum := uint64(0)
	for global != 0 {
		slice = global
		if slice > 64 {
			slice = 64
		}
		remaining := slice
		for remaining != 0 {
			captured := initial - global + slice - remaining
			sum += captured + 1
			remaining--
		}
		global -= slice - remaining
	}
	if sum != 501501 {
		print("FAIL\n")
		return 1
	}
	// A short cold exit/fault retires only the completed prefix, not the
	// difference between the aggregate session budget and a slice register.
	global = initial
	slice = 64
	remaining := uint64(45)
	global -= slice - remaining
	if initial-global != 19 || global != 982 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
