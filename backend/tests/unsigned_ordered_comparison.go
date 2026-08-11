package main

func appMain(args []string) int {
	// Keep values within the shared native positive range. This regression is
	// specifically for RTG condition selection (including unsigned <= and >=),
	// not the separate wide-uint ordering path used by 32-bit hosted targets.
	values := []uint32{0, 1, 2, 3, 4}
	for left := 0; left < len(values); left++ {
		for right := 0; right < len(values); right++ {
			less := values[left] < values[right]
			lessEqual := values[left] <= values[right]
			greater := values[left] > values[right]
			greaterEqual := values[left] >= values[right]
			if less != (left < right) || lessEqual != (left <= right) ||
				greater != (left > right) || greaterEqual != (left >= right) {
				print("FAIL\n")
				return 1
			}
		}
	}
	print("PASS\n")
	return 0
}
