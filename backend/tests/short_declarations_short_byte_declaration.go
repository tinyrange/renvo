package main

func appMain(args []string) int {
	x := byte('a')
	if !(x == 'a') {
		print("RENVO-0303 short_byte_declaration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
