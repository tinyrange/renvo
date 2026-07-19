package main

type Renvo0609Pair struct {
	a int
	b int
}

func appMain(args []string) int {
	total := 0
	for i := 0; i < 1; i = i + 1 {
		p := Renvo0609Pair{4, 5}
		total = p.a + p.b
	}
	if total != 9 {
		print("RENVO-0609 positional literal failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
