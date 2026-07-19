package main

type globalReturnColor struct {
	r byte
	g byte
	b byte
	a byte
}

var globalReturnValue = globalReturnColor{r: 12, g: 34, b: 56, a: 78}

func globalStructReturn() globalReturnColor {
	return globalReturnValue
}

func appMain() int {
	color := globalStructReturn()
	if color.r == 12 && color.g == 34 && color.b == 56 && color.a == 78 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
