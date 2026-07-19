package main

func appMain(args []string) int {
	x := 0
	for x < 4 {
		x = x + 1
	}
	if x != 4 {
		print("RENVO-0337 loop assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
