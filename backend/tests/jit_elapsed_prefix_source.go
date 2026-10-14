package main

// Supplemental parsed-source regression for capturing native dispatch time
// before reusing the remaining-budget register. The real failing generated-code
// regression is TestNativeElapsedExactLinkedPrefixAndLoopIteration.
func appMain(args []string) int {
	initial, remaining, total := 16, 16, 0
	clock := uint64(500)
	for body := 0; body < 2; body++ {
		captured := initial - remaining + total
		for iteration := 0; iteration < 2; iteration++ {
			at := clock + uint64(captured+iteration*4+1)*100
			want := uint64(600 + body*800 + iteration*400)
			if at != want {
				print("FAIL\n")
				return 1
			}
		}
		remaining -= 8
	}
	if remaining != 0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
