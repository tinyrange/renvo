package main

type rtgSecondSliceTok struct {
	kind int
}

type rtgSecondSliceProg struct {
	src  []byte
	toks []rtgSecondSliceTok
}

func rtgSecondSliceTokIsKind(p *rtgSecondSliceProg, i int, kind int) bool {
	return i >= 0 && i < len(p.toks) && p.toks[i].kind == kind
}

func appMain(args []string, env []string) int {
	var p rtgSecondSliceProg
	p.src = append(p.src, 'x')
	p.toks = append(p.toks, rtgSecondSliceTok{kind: 6})
	p.toks = append(p.toks, rtgSecondSliceTok{kind: 1})
	if rtgSecondSliceTokIsKind(&p, 1, 1) && !rtgSecondSliceTokIsKind(&p, 2, 1) {
		print("PASS\n")
	}
	return 0
}
