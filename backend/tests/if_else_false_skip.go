package main

func appMain(args []string) int {
	x := 4
	if false {
		x = 9
	}
	if x != 4 {
		print("RENVO-0352 if false failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
