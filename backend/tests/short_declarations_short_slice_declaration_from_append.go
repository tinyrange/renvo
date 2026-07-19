package main

func appMain(args []string) int {
	s := []int{}
	s = append(s, 5)
	if !(len(s) == 1 && s[0] == 5) {
		print("RENVO-0306 short_slice_declaration_from_append failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
