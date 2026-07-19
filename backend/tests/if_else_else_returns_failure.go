package main

func appMain(args []string) int {
	if 5 == 5 {
		print("PASS\n")
		return 0
	} else {
		print("RENVO-0370 early else failure\n")
		return 1
	}
}
