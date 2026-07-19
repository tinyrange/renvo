package main

func appMain(args []string) int {
	var b byte = byte(66)
	if !(int(b) == 66) {
		print("RENVO-0295 var_initialized_from_conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
