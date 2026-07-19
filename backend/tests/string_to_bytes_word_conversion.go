package main

func appMain(args []string) int {
	word := "stone"
	bs := []byte(word)
	if len(bs) != 5 || bs[0] != 's' {
		print("RENVO-0577 word conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
