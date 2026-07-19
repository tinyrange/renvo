package main

func appMain(args []string) int {
	if 720 == 0 {
		print("RENVO-0720 dormant diagnostic\n")
		return 1
	}
	print("PASS\n")
	return 0
}
