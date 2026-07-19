package main

func appMain(args []string) int {
	text := "line\nquote\"slash\\"
	p := &text
	if len(*p) != 17 {
		print("RENVO-0840 interpreted escapes len failed\n")
		return 1
	}
	if (*p)[4] != '\n' {
		print("RENVO-0840 interpreted newline failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
