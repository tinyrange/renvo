package main

type globalAliasColor struct {
	r byte
	g byte
	b byte
	a byte
}

var globalAliasWhite = globalAliasColor{r: 255, g: 254, b: 253, a: 252}
var globalAliasAccent = globalAliasWhite

func globalAliasForward(value globalAliasColor) globalAliasColor {
	return value
}

func appMain() int {
	value := globalAliasForward(globalAliasAccent)
	if value.r != 255 || value.g != 254 || value.b != 253 || value.a != 252 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
