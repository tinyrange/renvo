package main

func appMain(args []string) int {
	x := 1 +
		2*
			3 +
		4
	if x != 11 {
		print("RENVO-0814 multiline arithmetic failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
