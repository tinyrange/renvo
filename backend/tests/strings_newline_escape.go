package main

func appMain(args []string) int {
	s := "a\nb"
	if len(s) != 3 || s[1] != 10 {
		print("strings_03 newline\n")
		return 1
	}
	print("PASS\n")
	return 0
}
