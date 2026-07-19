package main

func appMain(args []string) int {
	s := []int{}
	s = append(s, 4)
	s = append(s, 5)
	s = append(s, 6)
	if !(s[1<<1] == 6) {
		print("RENVO-0244 shift_used_as_slice_index failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
