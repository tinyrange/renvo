package main

func appMain(args []string) int {
	if 3 > 4 {
		print("RENVO-0818 if branch failed\n")
		return 1
	} else {
		print("PASS\n")
		return 0
	}
}
