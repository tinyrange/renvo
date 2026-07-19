package main

type renvoSecondSliceTok struct {
	kind int
}

type renvoSecondSliceProg struct {
	src  []byte
	toks []renvoSecondSliceTok
}

func renvoSecondSliceTokIsKind(p *renvoSecondSliceProg, i int, kind int) bool {
	return i >= 0 && i < len(p.toks) && p.toks[i].kind == kind
}

func appMain(args []string, env []string) int {
	var p renvoSecondSliceProg
	p.src = append(p.src, 'x')
	p.toks = append(p.toks, renvoSecondSliceTok{kind: 6})
	p.toks = append(p.toks, renvoSecondSliceTok{kind: 1})
	if renvoSecondSliceTokIsKind(&p, 1, 1) && !renvoSecondSliceTokIsKind(&p, 2, 1) {
		print("PASS\n")
	}
	return 0
}
