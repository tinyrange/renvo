package main

var renvoFixedTarget int

func appMain() int {
	if renvoFixedTarget != 11 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
