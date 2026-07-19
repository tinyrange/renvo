package main

func appMain(args []string) int {
	xs := []int{0, 0, 0}
	index := 1
	switch index {
	case 0:
		xs[0] = 3
	case 1:
		xs[1] = 7
	default:
		xs[2] = 9
	}
	if xs[0] != 0 || xs[1] != 7 || xs[2] != 0 {
		print("RENVO-SWITCH-019 slice assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
