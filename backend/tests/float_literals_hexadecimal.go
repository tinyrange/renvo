package main

func appMain(args []string) int {
	a := 0x1.8p+1
	b := 0x1.4p+2
	c := 0x1.2p+1
	if a+b != 8.0 {
		print("float_literals_hexadecimal sum\n")
		return 1
	}
	if c != 2.25 {
		print("float_literals_hexadecimal frac\n")
		return 1
	}
	print("PASS\n")
	return 0
}
