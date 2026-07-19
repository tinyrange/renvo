package main

func appMain(args []string) int {
	ok := true
	x := 0
	if ok {
		x = 11
	}
	if x != 11 {
		print("RENVO-0361 bool condition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
