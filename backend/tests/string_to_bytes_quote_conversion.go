package main

func renvo0579Bytes(s string) []byte {
	return []byte(s)
}

func appMain(args []string) int {
	bs := renvo0579Bytes("a\"b")
	if len(bs) != 3 || bs[1] != '"' {
		print("RENVO-0579 quote conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
