package main

func appMain(args []string) int {
	x := 0
	for {
		if x == 3 {
			break
		}
		x = x + 1
	}
	if x != 3 {
		print("RENVO-0373 break control failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
