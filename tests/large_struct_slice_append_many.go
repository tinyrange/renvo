package main

type TokenLike struct {
	Kind   string
	Text   string
	Start  int
	End    int
	Line   int
	Column int
	Extra  int
	More   int
}

func makeToken(i int) TokenLike {
	return TokenLike{Kind: "ident", Text: "package", Start: i, End: i + 7, Line: 1, Column: 1, Extra: i + 1, More: i + 2}
}

func appMain(args []string) int {
	var toks []TokenLike
	i := 0
	for i < 5000 {
		tok := makeToken(i)
		toks = append(toks, tok)
		i += 1
	}
	if len(toks) != 5000 {
		print("FAIL\n")
		return 1
	}
	if toks[0].Text != "package" || toks[0].Start != 0 || toks[4999].Start != 4999 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
