package main

func appMain(args []string) int {
	x := 13
	goto target
target:
	if x != 13 {
		print("RENVO-0473 app target failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
