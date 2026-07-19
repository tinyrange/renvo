package main

func appMain(args []string) int {
	x := 8
	goto keep
	x = 0
keep:
	if x != 8 {
		print("RENVO-0465 preserve local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
