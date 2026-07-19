package main

func strings13(s string) bool { return len(s) == 5 && s[0] == 'h' }
func appMain(args []string) int {
	if !strings13("hello") {
		print("strings_13 param\n")
		return 1
	}
	print("PASS\n")
	return 0
}
