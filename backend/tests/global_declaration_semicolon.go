package main

var value int32 = 7;

func appMain() int {
	if value != 7 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
