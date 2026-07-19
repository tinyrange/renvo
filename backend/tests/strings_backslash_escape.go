package main

func appMain(args []string) int {
	s := "a\\b"
	if len(s) != 3 || s[1] != 92 {
		print("strings_05 slash\n")
		return 1
	}
	print("PASS\n")
	return 0
}
