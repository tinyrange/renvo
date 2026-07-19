package main

func appMain(args []string) int {
	if !(1+2<<3 == 17) {
		print("RENVO-0240 addition_after_shift_precedence failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
