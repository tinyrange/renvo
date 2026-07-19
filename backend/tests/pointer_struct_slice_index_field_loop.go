package main

type miniTok struct {
	kind  int
	start int
	end   int
	line  int
}

type miniProgram struct {
	src  []byte
	toks []miniTok
}

func miniStatementEnd(p *miniProgram, start int, end int) int {
	line := p.toks[start].line
	i := start
	paren := 0
	for i < end {
		if p.toks[i].kind == '(' {
			paren++
		} else if p.toks[i].kind == ')' {
			paren--
		}
		if i > start && p.toks[i].line != line && paren == 0 {
			return i
		}
		i++
	}
	return i
}

func appMain() int {
	var p miniProgram
	p.src = []byte("print(\"PASS\\n\")\nnext")
	p.toks = make([]miniTok, 0, 8)
	p.toks = append(p.toks, miniTok{kind: 1, start: 0, end: 5, line: 3})
	p.toks = append(p.toks, miniTok{kind: '(', start: 5, end: 6, line: 3})
	p.toks = append(p.toks, miniTok{kind: 2, start: 6, end: 14, line: 3})
	p.toks = append(p.toks, miniTok{kind: ')', start: 14, end: 15, line: 3})
	p.toks = append(p.toks, miniTok{kind: 3, start: 16, end: 20, line: 4})
	end := miniStatementEnd(&p, 0, len(p.toks))
	if end == 4 && p.toks[0].line == 3 && p.toks[4].line == 4 {
		print("PASS\n")
	}
	return 0
}
