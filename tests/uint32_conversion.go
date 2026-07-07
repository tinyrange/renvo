package main

func appMain() int {
	v := uint32(1000)
	if int(v) == 1000 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
