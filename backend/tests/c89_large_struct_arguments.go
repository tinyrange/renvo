package main

type c89LargeStructArguments struct {
	a int
	b int
	c int
	d int
	e int
	f int
	g int
	h int
	i int
}

func c89LargeStructArgumentsValid(value c89LargeStructArguments, tail int) bool {
	return value.a == 1 && value.b == 2 && value.c == 3 &&
		value.d == 4 && value.e == 5 && value.f == 6 &&
		value.g == 7 && value.h == 8 && value.i == 9 && tail == 10
}

func c89LargeStructArgumentsSelect(
	first c89LargeStructArguments,
	second c89LargeStructArguments,
) c89LargeStructArguments {
	if first.a != 1 || first.e != 5 || first.i != 9 ||
		second.a != 11 || second.e != 15 || second.i != 19 {
		return c89LargeStructArguments{}
	}
	return second
}

func appMain() int {
	value := c89LargeStructArguments{a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9}
	if !c89LargeStructArgumentsValid(value, 10) {
		return 1
	}
	second := c89LargeStructArguments{a: 11, b: 12, c: 13, d: 14, e: 15, f: 16, g: 17, h: 18, i: 19}
	selected := c89LargeStructArgumentsSelect(value, second)
	if selected.a != 11 || selected.e != 15 || selected.i != 19 {
		return 2
	}
	print("PASS\n")
	return 0
}
