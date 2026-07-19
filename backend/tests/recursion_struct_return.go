package main

type Renvo0518Pair struct {
	a int
	b int
}

func renvo0518Make(n int) Renvo0518Pair {
	if n == 0 {
		return Renvo0518Pair{a: 1, b: 1}
	}
	p := renvo0518Make(n - 1)
	return Renvo0518Pair{a: p.a + 1, b: p.b + int(byte(n))}
}

func appMain(args []string) int {
	p := renvo0518Make(3)
	if p.a != 4 || p.b != 7 {
		print("RENVO-0518 struct return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
