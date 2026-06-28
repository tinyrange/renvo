package main

type Token struct {
	Kind   string
	Text   string
	Start  int
	End    int
	Line   int
	Column int
}

type ImportInfo struct {
	Path   string
	Alias  string
	Line   int
	Column int
}

func tokens() ([]Token, error) {
	var toks []Token
	tok := Token{Kind: "string", Text: "fmt", Line: 1, Column: 1}
	toks = append(toks, tok)
	return toks, nil
}

func makeImports() []ImportInfo {
	toks, _ := tokens()
	var imports []ImportInfo
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		info := ImportInfo{Path: tok.Text, Line: tok.Line, Column: tok.Column}
		imports = append(imports, info)
	}
	return imports
}

func appMain(args []string, env []string) int {
	imports := makeImports()
	if len(imports) != 1 {
		print("FAIL len\n")
		return 1
	}
	first := imports[0]
	if first.Path != "fmt" {
		print("FAIL first\n")
		return 1
	}
	print("PASS\n")
	return 0
}
