package main

func appMain(args []string) int {
	s := "a\"b"
	if len(s) != 3 || s[1] != 34 {
		print("strings_04 quote\n")
		return 1
	}
	print("PASS\n")
	return 0
}
