package main

func appMain(args []string) int {
	x := 4
	if false {
		x = 0
	} else {
		x := 8
		if x != 8 {
			print("RENVO-0366 inner failed\n")
			return 1
		}
	}
	if x != 4 {
		print("RENVO-0366 shadow else failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
