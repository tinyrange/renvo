package main

func appMain(args []string) int {
	x := 4
	if x != 4 {
		print("RENVO-0802 tabs failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
