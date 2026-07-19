package main

type Renvo0617Inner struct{ value int }
type Renvo0617Outer struct{ inner Renvo0617Inner }

func appMain(args []string) int {
	var xs []int
	outer := Renvo0617Outer{inner: Renvo0617Inner{value: 10}}
	xs = append(xs, outer.inner.value)
	if len(xs) != 1 || xs[0] != 10 {
		print("RENVO-0617 nested field read failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
