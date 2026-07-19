package main

func appMain(args []string) int {
	s := "a\n\\b" // escapes before comment
	if len(s) != 4 {
		print("RENVO-0824 escape length failed\n")
		return 1
	}
	if s[1] != '\n' {
		print("RENVO-0824 newline escape failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
