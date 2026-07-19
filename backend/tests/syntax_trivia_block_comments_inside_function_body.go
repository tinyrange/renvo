package main

func appMain(args []string) int {
	/* block comment inside function */
	x := 8
	if x != 8 {
		print("RENVO-0808 body block comment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
