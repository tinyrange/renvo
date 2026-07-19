package main

func appMain(args []string) int {
	var s []byte
	s = append(s, 65)
	if !(len(s) == 1 && s[0] == 65) {
		print("RENVO-0283 var_byte_zero_value_then_append failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
