package main

type globalCallColor struct {
	r byte
	g byte
	b byte
	a byte
}

func globalCallRGBA(r byte, g byte, b byte, a byte) globalCallColor {
	return globalCallColor{r: r, g: g, b: b, a: a}
}

var globalCallBlack = globalCallRGBA(0, 1, 2, 255)

func appMain(args []string) int {
	if globalCallBlack.r == 0 && globalCallBlack.g == 1 && globalCallBlack.b == 2 && globalCallBlack.a == 255 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
