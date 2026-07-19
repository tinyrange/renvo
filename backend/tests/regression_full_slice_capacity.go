package main

func appMain() int {
	xs := []int{1, 2, 3, 4}
	ys := xs[1:2:2]
	ys = append(ys, 9)
	if len(ys) != 2 {
		print("RENVO-1150 full slice append length failed\n")
		return 1
	}
	if ys[0] != 2 || ys[1] != 9 {
		print("RENVO-1150 full slice append values failed\n")
		return 1
	}
	if xs[2] != 3 {
		print("RENVO-1150 full slice append capacity failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
