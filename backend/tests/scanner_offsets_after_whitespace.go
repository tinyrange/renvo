package main

type miniToken struct {
	kind  string
	text  string
	start int
	end   int
	line  int
	col   int
}

func miniIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func miniIdent(c byte) bool {
	return miniIdentStart(c) || (c >= '0' && c <= '9')
}

func miniTokens(src []byte) []miniToken {
	var toks []miniToken
	line := 1
	col := 1
	i := 0
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' {
			i++
			col++
			continue
		}
		if c == '\n' {
			i++
			line++
			col = 1
			continue
		}
		if miniIdentStart(c) {
			start := i
			startLine := line
			startCol := col
			i++
			col++
			for i < len(src) && miniIdent(src[i]) {
				i++
				col++
			}
			toks = append(toks, miniToken{kind: "ident", text: string(src[start:i]), start: start, end: i, line: startLine, col: startCol})
			continue
		}
		i++
		col++
	}
	return toks
}

func appMain(args []string, env []string) int {
	src := []byte(" \n\tIdent = value\n  Next")
	toks := miniTokens(src)
	if len(toks) != 3 {
		print("bad token count\n")
		return 1
	}
	if toks[0].kind != "ident" || toks[0].text != "Ident" || toks[0].start != 3 || toks[0].end != 8 || toks[0].line != 2 || toks[0].col != 2 {
		print("bad first token\n")
		return 1
	}
	if toks[1].line != 2 || toks[1].col != 10 || string(src[toks[1].start:toks[1].end]) != "value" {
		print("bad second span\n")
		return 1
	}
	if toks[2].text != "Next" || string(src[toks[2].start:toks[2].end]) != "Next" {
		print("bad third token\n")
		return 1
	}
	print("PASS\n")
	return 0
}
