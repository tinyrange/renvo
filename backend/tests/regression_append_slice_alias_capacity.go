package main

func appMain() int {
	xs := []int{10, 20, 30}
	ys := xs[:1]
	arg := xs[1]
	next := append(ys, 99)
	ys = append(next, arg)
	if len(ys) != 3 {
		print("RENVO-1140 append alias length failed\n")
		return 1
	}
	if ys[1] != 99 || ys[2] != 20 {
		print("RENVO-1140 append alias values failed\n")
		return 1
	}
	if xs[1] != 99 {
		print("RENVO-1140 append alias backing failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
