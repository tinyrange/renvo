package main

func appMain() int {
	var value uint32 = 4000000000
	if uint64(value) != 4000000000 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
