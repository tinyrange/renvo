package main

func appMain(args []string) int {
	s := "cat"
	if s[1] != 'a' {
		print("strings_07 middle\n")
		return 1
	}
	print("PASS\n")
	return 0
}
