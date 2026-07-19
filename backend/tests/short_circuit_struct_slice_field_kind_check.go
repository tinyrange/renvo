package main

type renvoShortCircuitTok struct {
	kind int
}

type renvoShortCircuitProg struct {
	toks []renvoShortCircuitTok
}

func renvoShortCircuitTokIsKind(p *renvoShortCircuitProg, i int, kind int) bool {
	return i >= 0 && i < len(p.toks) && p.toks[i].kind == kind
}

func appMain(args []string, env []string) int {
	var p renvoShortCircuitProg
	p.toks = append(p.toks, renvoShortCircuitTok{kind: 6})
	p.toks = append(p.toks, renvoShortCircuitTok{kind: 1})
	if renvoShortCircuitTokIsKind(&p, 1, 1) && !renvoShortCircuitTokIsKind(&p, 2, 1) {
		print("PASS\n")
	}
	return 0
}
