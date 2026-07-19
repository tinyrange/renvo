package main

type renvo0488Pair struct {
	a int
	b int
}

func renvo0488Make() renvo0488Pair { return renvo0488Pair{a: 4, b: 5} }
func appMain(args []string) int {
	p := renvo0488Make()
	if p.a+p.b != 9 {
		print("RENVO-0488 struct return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
