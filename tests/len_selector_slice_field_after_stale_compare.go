package main

type rtgLenSelectorProg struct {
	src  []byte
	toks []int
}

func rtgLenSelectorLen(p *rtgLenSelectorProg) int {
	a := 0
	b := 0
	if a == b {
	}
	return len(p.toks)
}

func appMain(args []string, env []string) int {
	var p rtgLenSelectorProg
	p.src = append(p.src, 'x')
	p.toks = append(p.toks, 10)
	p.toks = append(p.toks, 20)
	if rtgLenSelectorLen(&p) == 2 {
		print("PASS\n")
	}
	return 0
}
