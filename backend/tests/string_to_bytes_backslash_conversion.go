package main

func appMain(args []string) int {
	bs := renvo0580Bytes("a\\b")
	if len(bs) != 3 || bs[1] != '\\' {
		print("RENVO-0580 backslash conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func renvo0580Bytes(s string) []byte {
	return []byte(s)
}
