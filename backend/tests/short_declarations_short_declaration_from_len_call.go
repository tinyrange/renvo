package main

func appMain(args []string) int {
	s := []byte("abc")
	n := len(s)
	if !(n == 3) {
		print("RENVO-0320 short_declaration_from_len_call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
