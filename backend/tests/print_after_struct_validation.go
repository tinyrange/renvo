package main

type renvo0714Pair struct {
	a int
	b int
}

func appMain(args []string) int {
	p := renvo0714Pair{a: 3, b: 8}
	if p.a+p.b != 11 {
		print("RENVO-0714 struct diagnostic failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
