package main

type wideScratchNamedSlice []int64
type wideScratchNamedMap map[string]int64

func wideScratchDisturbance() int64 {
	values := []int64{0, 0, 0, 0, 0}
	window := values[:1]
	_ = window
	return 0
}

func wideScratchVariadic(base int64, values ...int64) int64 {
	total := base
	for index, value := range values {
		total += int64(index+1) * value
	}
	return total
}

func wideScratchValue() int64 {
	values := []int64{}
	return wideScratchVariadic(-0, values...) + wideScratchVariadic(0, -0, 0)
}

func appMain(args []string) int {
	if wideScratchDisturbance() != 0 {
		print("FAIL: scratch disturbance\n")
		return 1
	}
	if wideScratchValue() != 0 {
		print("FAIL: uint64 scratch type identity\n")
		return 1
	}
	print("PASS\n")
	return 0
}
