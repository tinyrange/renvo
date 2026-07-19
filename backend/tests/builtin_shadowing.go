package main

type builtinShadowValue struct {
	n int
}

func copy(value builtinShadowValue) int {
	return value.n
}

func appMain() int {
	if copy(builtinShadowValue{n: 7}) == 7 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
