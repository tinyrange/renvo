package main

func appMain(args []string) int {
	if !("renvo" != "go") {
		print("RENVO-0194 string_inequality failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
