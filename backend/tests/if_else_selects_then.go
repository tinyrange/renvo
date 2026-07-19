package main

func appMain(args []string) int {
	x := 0
	if 2 < 3 {
		x = 7
	} else {
		x = 8
	}
	if x != 7 {
		print("RENVO-0353 then branch failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
