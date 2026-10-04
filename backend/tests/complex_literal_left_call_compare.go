package main

func complexLiteralRight() complex64 {
	return complex64(1 + 2i)
}

func appMain() int {
	if 1+2i != complexLiteralRight() {
		return 1
	}
	print("PASS\n")
	return 0
}
