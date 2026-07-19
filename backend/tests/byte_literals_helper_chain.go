package main

func byteLit24a(b byte) byte { return byteLit24b(b) }
func byteLit24b(b byte) byte { return byte(int(b) + 1) }
func appMain(args []string) int {
	if byteLit24a('V') != 'W' {
		print("byte_literals_24 chain\n")
		return 1
	}
	print("PASS\n")
	return 0
}
