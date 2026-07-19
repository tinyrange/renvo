package main

func appMain(args []string) int {
	s := "cat"
	if s[0] != 'c' {
		print("strings_06 first\n")
		return 1
	}
	print("PASS\n")
	return 0
}
