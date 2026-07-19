package main

func appMain(args []string) int {
	x := 0
	if false {
		x = 5
	} else if false {
		x = 6
	} else {
		x = 7
	}
	if x != 7 {
		print("RENVO-0357 else-if else failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
