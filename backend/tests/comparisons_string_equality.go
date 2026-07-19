package main

func appMain(args []string) int {
	if !("renvo" == "renvo") {
		print("RENVO-0193 string_equality failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
