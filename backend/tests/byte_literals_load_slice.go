package main

func appMain(args []string) int {
	b := []byte("GH")
	if b[0] != 'G' {
		print("byte_literals_12 load\n")
		return 1
	}
	print("PASS\n")
	return 0
}
