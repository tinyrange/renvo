package main

type Callback func() int
type Factory func(int) Callback

func invoke(cb func()) { cb() }

func recursiveNested() int {
	total := 0
	var visit func(int)
	visit = func(index int) {
		total += index
		if index > 0 {
			invoke(func() { visit(index - 1) })
		}
	}
	visit(4)
	return total
}

func makeFactory(base int) Factory {
	return func(index int) Callback {
		local := index * 2
		return func() int {
			local++
			return base + index + local
		}
	}
}

func main() {
	factory := makeFactory(10)
	a := factory(3)
	b := factory(5)
	if a() != 20 || a() != 21 || b() != 26 {
		panic("nested captured lifetime")
	}
	if recursiveNested() != 10 {
		panic("nested recursion")
	}
	println("PASS")
}
