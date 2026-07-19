package main

type Renvo0620Pair struct {
	a int
	b int
}

func renvo0620Make(ok bool) Renvo0620Pair {
	if ok && true {
		return Renvo0620Pair{a: 8, b: 1}
	}
	return Renvo0620Pair{a: 0, b: 0}
}

func appMain(args []string) int {
	p := renvo0620Make(true)
	if p.a-p.b != 7 {
		print("RENVO-0620 struct return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
