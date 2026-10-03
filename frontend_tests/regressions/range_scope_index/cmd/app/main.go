package main

type Count int

func values[V any](items []V) func(func(V) bool) {
	return func(yield func(V) bool) {
		for _, item := range items {
			if !yield(item) {
				return
			}
		}
	}
}

func main() {
	total := 0
	for n := range values([]Count{1, 2, 3}) {
		visit := func(items []int) {
			for _, item := range items {
				total += item
			}
			for m := range n {
				total += int(m)
			}
			{
				items := Count(2)
				for m := range items {
					total += int(m)
				}
			}
			for _, item := range items {
				total += item
			}
		}
		visit([]int{int(n)})
	}
	for _, n := range []int{1, 2} {
		total += n
	}
	for n := range Count(2) {
		total += int(n)
	}
	if total != 23 {
		panic("range scopes")
	}
	println("PASS")
}
