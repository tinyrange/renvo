package main

func appMain() int {
	decoded := []rune("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if len(decoded) != 128 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
