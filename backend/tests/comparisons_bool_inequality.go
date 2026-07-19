package main

func appMain(args []string) int {
	if !(true != false) {
		print("RENVO-0196 bool_inequality failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
