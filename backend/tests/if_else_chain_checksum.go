package main

func appMain(args []string) int {
	x := 6
	score := 0
	if x < 3 {
		score = 1
	} else if x < 6 {
		score = 2
	} else if x == 6 {
		score = 13
	} else {
		score = 4
	}
	if score != 13 {
		print("RENVO-0375 if chain checksum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
