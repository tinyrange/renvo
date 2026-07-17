package main

var arraySlicingBoundsRecovered int

func arraySlicingBoundsFailure(mode int) {
	defer func() {
		if recover() != nil {
			arraySlicingBoundsRecovered++
			return
		}
		print("FAIL\n")
	}()
	values := [3]int{1, 2, 3}
	if mode == 0 {
		low := -1
		_ = values[low:]
	}
	if mode == 1 {
		high := 4
		_ = values[:high]
	}
	if mode == 2 {
		low := 2
		high := 1
		_ = values[low:high]
	}
	if mode == 3 {
		high := 2
		max := 1
		_ = values[0:high:max]
	}
	if mode == 4 {
		max := 4
		_ = values[0:2:max]
	}
	print("FAIL\n")
}

func appMain(args []string) int {
	for mode := 0; mode < 5; mode++ {
		arraySlicingBoundsFailure(mode)
	}
	if arraySlicingBoundsRecovered != 5 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
