package main

func appMain(args []string) int {
	text := "stable"
	bs := []byte(text)
	bs[0] = 'S'
	if len(text) != 6 {
		print("RENVO-0599 original string length failed\n")
		return 1
	}
	if text[0] != 's' {
		print("RENVO-0599 original string changed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
