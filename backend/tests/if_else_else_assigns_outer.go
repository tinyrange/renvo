package main

func appMain(args []string) int {
	x := 0
	if false {
		x = 1
	} else {
		x = 18
	}
	if x != 18 {
		print("RENVO-0368 outer else assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
