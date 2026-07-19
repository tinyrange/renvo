package main

func appMain(args []string) int {
	x := 5
	p := &x
	*p += 6
	if x != 11 {
		print("RENVO-0342 pointer compound failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
