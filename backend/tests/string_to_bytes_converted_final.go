package main

func appMain(args []string) int {
	bs := []byte("pass")
	if len(bs) != 4 || bs[3] != 's' {
		print("RENVO-0600 converted final failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
