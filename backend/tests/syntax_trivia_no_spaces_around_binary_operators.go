package main

func appMain(args []string) int {
	if 1+2*3 != 7 {
		print("RENVO-0804 tight operators failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
