package main

type Renvo0619Pair struct {
	a int
	b int
}

func renvo0619Sum(p Renvo0619Pair) int {
	total := p.a
	total += p.b
	return total
}

func appMain(args []string) int {
	if renvo0619Sum(Renvo0619Pair{a: 3, b: 4}) != 7 {
		print("RENVO-0619 struct by value failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
