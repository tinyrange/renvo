package main

type stageToken struct {
	Kind   string
	Text   string
	Start  int
	End    int
	Line   int
	Column int
	Extra  int
	More   int
}

func collectStageTokens(limit int) []stageToken {
	toks := make([]stageToken, 0, 25000)
	i := 0
	for i < limit {
		toks = append(toks, stageToken{Kind: "ident", Text: "package", Start: i, End: i + 7, Line: 1, Column: 1, Extra: i + 1, More: i + 2})
		i++
	}
	return toks
}

func appMain(args []string) int {
	toks := collectStageTokens(25000)
	if len(toks) != 25000 {
		print("FAIL\n")
		return 1
	}
	if toks[0].Text != "package" || toks[0].Start != 0 {
		print("FAIL\n")
		return 1
	}
	if toks[24999].Text != "package" || toks[24999].Start != 24999 || toks[24999].End != 25006 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
