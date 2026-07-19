package main

func appMain(args []string) int {
	if !(true == true) {
		print("RENVO-0195 bool_equality failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
