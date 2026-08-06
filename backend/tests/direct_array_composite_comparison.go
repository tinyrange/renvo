package main

func appMain() int {
	if [2]int64{3, 5} != [2]int64{3, 5} {
		print("FAIL: direct array composite comparison\n")
		return 1
	}
	print("PASS\n")
	return 0
}
