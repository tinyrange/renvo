package main

type Renvo0534Pair struct {
	a int
	b bool
}

func renvo0534Make() Renvo0534Pair {
	return Renvo0534Pair{9, true}
}

func appMain(args []string) int {
	for i := 0; i < 1; i = i + 1 {
		p := renvo0534Make()
		if p.a != 9 || !p.b {
			print("RENVO-0534 struct return failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
