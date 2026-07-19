package main

func appMain(args []string) int {
	x := 0
	if true {
		x = 3
	}
	if x != 3 {
		print("RENVO-0351 if true failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
