package main

func renvo0531Byte(s string, i int) byte {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func appMain(args []string) int {
	if renvo0531Byte("axis", 1) != 'x' {
		print("RENVO-0531 byte return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
