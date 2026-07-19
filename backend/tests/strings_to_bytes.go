package main

func appMain(args []string) int {
	b := []byte("bytes")
	if len(b) != 5 || b[0] != 'b' || b[4] != 's' {
		print("strings_25 bytes\n")
		return 1
	}
	print("PASS\n")
	return 0
}
