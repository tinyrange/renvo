package main

type globalArgumentColor struct {
	r byte
	g byte
	b byte
	a byte
}

var globalArgumentWhite = globalArgumentColor{r: 255, g: 254, b: 253, a: 252}

func globalArgumentMatches(c globalArgumentColor) bool {
	return c.r == 255 && c.g == 254 && c.b == 253 && c.a == 252
}

func appMain(args []string) int {
	if !globalArgumentMatches(globalArgumentWhite) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
