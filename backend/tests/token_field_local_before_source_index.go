package main

type tokenFieldLocalTok struct {
	kind  int
	start int
	end   int
	line  int
}

type tokenFieldLocalProgram struct {
	src  []byte
	toks []tokenFieldLocalTok
}

func tokenFieldLineContinues(p *tokenFieldLocalProgram, i int) bool {
	if i <= 0 {
		return false
	}
	prev := i - 1
	tokStart := p.toks[prev].start
	tokEnd := p.toks[prev].end
	if tokEnd <= tokStart {
		return false
	}
	c := p.src[tokStart]
	return c == '+'
}

func appMain() int {
	var p tokenFieldLocalProgram
	p.src = []byte("abc+def")
	p.toks = make([]tokenFieldLocalTok, 0, 4)
	p.toks = append(p.toks, tokenFieldLocalTok{kind: 1, start: 3, end: 4, line: 1})
	p.toks = append(p.toks, tokenFieldLocalTok{kind: 2, start: 4, end: 7, line: 1})
	if tokenFieldLineContinues(&p, 1) {
		print("PASS\n")
	}
	return 0
}
