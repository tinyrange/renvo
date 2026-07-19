package main

func appMain(args []string) int {
	s := "cat"
	if s[2] != 't' {
		print("strings_08 last\n")
		return 1
	}
	print("PASS\n")
	return 0
}
