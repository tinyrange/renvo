package main

func appMain() int {
	array := &[4]int64{2, 3, 5, 7}
	window := array[1:3]
	if len(window) != 2 || cap(window) != 3 || window[0] != 3 || window[1] != 5 {
		print("FAIL: pointer array composite\n")
		return 1
	}
	print("PASS\n")
	return 0
}
