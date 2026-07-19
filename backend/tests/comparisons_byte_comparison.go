package main

func appMain(args []string) int {
	var b byte = 'q'
	if !(b == 'q') {
		print("RENVO-0192 byte_comparison failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
