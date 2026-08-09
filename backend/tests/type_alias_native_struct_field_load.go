package main

type renvoAliasPackedColor struct {
	r byte
	g byte
	b byte
	a byte
}

type renvoAliasPackedColorAlias = renvoAliasPackedColor

func renvoAliasPackedColorValue() renvoAliasPackedColorAlias {
	return renvoAliasPackedColorAlias{r: 27, g: 31, b: 38, a: 255}
}

func appMain(args []string) int {
	got := renvoAliasPackedColorValue()
	if got.r != 27 || got.g != 31 || got.b != 38 || got.a != 255 {
		print("type alias native struct field load failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
