package main

type renvo0674Count int

func appMain(args []string) int {
	x := renvo0674Count(6)
	p := &x
	*p = *p + renvo0674Count(5)
	if int(x) != 11 {
		print("RENVO-0674 named pointer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
