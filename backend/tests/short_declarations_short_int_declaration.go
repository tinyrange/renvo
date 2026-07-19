package main

func appMain(args []string) int {
	x := 12
	if !(x == 12) {
		print("RENVO-0301 short_int_declaration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
