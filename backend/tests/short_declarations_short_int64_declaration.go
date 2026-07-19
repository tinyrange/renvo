package main

func appMain(args []string) int {
	var base int64 = 12
	x := base + 1
	if !(x == 13) {
		print("RENVO-0302 short_int64_declaration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
