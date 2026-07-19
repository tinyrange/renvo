package main

func appMain(args []string) int {
	x := 3
	if true {
		x := 9
		if x != 9 {
			print("RENVO-0365 inner failed\n")
			return 1
		}
	}
	if x != 3 {
		print("RENVO-0365 shadow then failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
