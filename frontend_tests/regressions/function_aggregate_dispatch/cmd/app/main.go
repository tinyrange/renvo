package main

type Holder struct{ F func(int) int }

func plus(x int) int  { return x + 1 }
func other(x int) int { return x + 100 }

var immediate = func(x int) int { return x + 2 }(3)
var global = Holder{F: plus}

func mutate(h *Holder) int { h.F = other; return 5 }
func factory() Holder      { return Holder{F: plus} }

func counter(start int) Holder {
	n := start
	return Holder{F: func(x int) int { n += x; return n }}
}

func nilCall(mode int) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	var h Holder
	if mode == 0 {
		_ = h.F(1)
	}
	if mode == 1 {
		defer h.F(1)
	}
	if mode == 2 {
		m := map[string]func(int) int{}
		_ = m["missing"](1)
	}
	return
}

func main() {
	m := map[string]func(int) int{"plus": plus}
	h := Holder{F: plus}
	n := 3
	c := Holder{F: func(x int) int { return x + n }}
	if m["plus"](1) != 2 || h.F(2) != 3 || c.F(4) != 7 || immediate != 5 || global.F(5) != 6 {
		panic("aggregate")
	}
	n = 4
	if c.F(4) != 8 {
		panic("shared closure environment")
	}
	callbacks := map[string]func(int) int{"captured": func(x int) int { return x + n }}
	n = 5
	if callbacks["captured"](4) != 9 {
		panic("map closure cell")
	}
	if h.F(mutate(&h)) != 6 || h.F(5) != 105 {
		panic("callee snapshot")
	}
	if factory().F(9) != 10 {
		panic("temporary field")
	}
	if !nilCall(0) || !nilCall(1) || !nilCall(2) {
		panic("nil dispatch")
	}
	uncaptured := Holder{F: func(x int) int { return x * 2 }}
	if uncaptured.F(4) != 8 {
		panic("uncaptured literal")
	}
	first := counter(10)
	copyOfFirst := first
	second := counter(20)
	if first.F(1) != 11 || copyOfFirst.F(2) != 13 || second.F(1) != 21 {
		panic("escaping shared cells")
	}
	println("PASS")
}
