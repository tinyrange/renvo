package main

type Renvo0564Box struct {
	total int
}

func renvo0564Sum(xs []int) Renvo0564Box {
	total := 0
	for i := 0; i < len(xs); i = i + 1 {
		total = total + xs[i]
	}
	return Renvo0564Box{total: total}
}

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 5)
	xs = append(xs, 6)
	if renvo0564Sum(xs).total != 11 {
		print("RENVO-0564 slice helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
