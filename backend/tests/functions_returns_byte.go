package main

func renvo0486Byte() byte { return byte(90) }
func appMain(args []string) int {
	if renvo0486Byte() != byte(90) {
		print("RENVO-0486 byte return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
