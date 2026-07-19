package main

func shortHelp() int {
	return 31
}
func appMain(args []string) int {
	x := shortHelp()
	if !(x == 31) {
		print("RENVO-0309 short_declaration_from_function_call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
