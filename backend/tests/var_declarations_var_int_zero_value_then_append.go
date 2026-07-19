package main

func appMain(args []string) int {
	var s []int
	s = append(s, 7)
	if !(len(s) == 1 && s[0] == 7) {
		print("RENVO-0284 var_int_zero_value_then_append failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
