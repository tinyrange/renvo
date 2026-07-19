package main

func appMain(args []string) int {
	x := 2
	goto work
work:
	x *= 5
	if x != 10 {
		print("RENVO-0471 goto compound failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
