package main

func renvo1005Text() (string, byte) {
	s := "go"
	return s, s[1]
}

func appMain(args []string) int {
	s, b := renvo1005Text()
	if s != "go" || b != 'o' {
		print("RENVO-1005 string byte pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
