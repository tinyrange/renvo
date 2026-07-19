package main

func sumDownInt(n int) int {
	if n == 0 {
		return 0
	}
	return n + sumDownInt(n-1)
}
func sumDownByte(b byte) int {
	if b == 0 {
		return 0
	}
	return int(b) + sumDownByte(byte(int(b)-1))
}

func appMain(args []string) int {
	if sumDownInt(4) != 10 {
		print("RENVO-0838 int helper recursion failed\n")
		return 1
	}
	if sumDownByte(byte(3)) != 6 {
		print("RENVO-0838 byte helper recursion failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
