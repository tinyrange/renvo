package main

type renvo0812Pair struct {
	a int
	b int
}

func appMain(args []string) int {
	p := renvo0812Pair{
		a: 6,
		b: 7,
	}
	if p.a+p.b != 13 {
		print("RENVO-0812 multiline literal failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
