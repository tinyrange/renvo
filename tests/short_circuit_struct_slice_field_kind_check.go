package main

type rtgShortCircuitTok struct {
	kind int
}

type rtgShortCircuitProg struct {
	toks []rtgShortCircuitTok
}

func rtgShortCircuitTokIsKind(p *rtgShortCircuitProg, i int, kind int) bool {
	return i >= 0 && i < len(p.toks) && p.toks[i].kind == kind
}

func appMain(args []string, env []string) int {
	var p rtgShortCircuitProg
	p.toks = append(p.toks, rtgShortCircuitTok{kind: 6})
	p.toks = append(p.toks, rtgShortCircuitTok{kind: 1})
	if rtgShortCircuitTokIsKind(&p, 1, 1) && !rtgShortCircuitTokIsKind(&p, 2, 1) {
		print("PASS\n")
	}
	return 0
}
