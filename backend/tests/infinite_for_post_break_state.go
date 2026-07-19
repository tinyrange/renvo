package main

func appMain(args []string) int {
	x := 0
	for {
		x = x + 1
		if x == 2 {
			break
		}
	}
	x = x + 5
	if x != 7 {
		print("RENVO-0449 post break state failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
