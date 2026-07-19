package main

func appMain(args []string) int {
	x := 29
	x %= 6
	if x != 5 {
		print("RENVO-0335 modulo compound failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
