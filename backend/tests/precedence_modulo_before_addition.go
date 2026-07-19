package main

func appMain(args []string) int {
	if !(10+11%4 == 13) {
		print("RENVO-0253 modulo_before_addition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
