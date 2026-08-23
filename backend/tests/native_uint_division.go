package main

func appMain() int {
	left := uint(0xf0000000)
	right := uint(31)
	if left/right != uint(129888123) || left%right != uint(27) {
		print("FAIL: native uint division\n")
		return 1
	}
	print("PASS\n")
	return 0
}
