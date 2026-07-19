package main

func appMain(args []string) int {
	x := 0
	if 4 < 3 {
		x = 7
	} else {
		x = 8
	}
	if x != 8 {
		print("RENVO-0354 else branch failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
