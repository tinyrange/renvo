package main

func rtg1005Text() (string, byte) {
	s := "go"
	return s, s[1]
}

func appMain(args []string) int {
	s, b := rtg1005Text()
	if s != "go" || b != 'o' {
		print("RTG-1005 string byte pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
