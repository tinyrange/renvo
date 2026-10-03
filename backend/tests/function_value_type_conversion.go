package main

type ConvertedFunction func(int) int

func convertedDirect(v int) int { return v + 2 }

type convertedReceiver struct{ Delta int }

func (r convertedReceiver) Add(v int) int { return v + r.Delta }

func appMain() int {
	shift := 2
	f := func(v int) int { return v + shift }
	g := ConvertedFunction(f)
	if g(40) != 42 {
		return 1
	}
	shift = 3
	if g(39) != 42 {
		return 2
	}
	h := ConvertedFunction(convertedDirect)
	if h(40) != 42 {
		return 3
	}
	r := convertedReceiver{2}
	m := ConvertedFunction(r.Add)
	r.Delta = 9
	if m(40) != 42 {
		return 4
	}
	type Local = ConvertedFunction
	u := Local(g)
	if u(39) != 42 {
		return 5
	}
	zero := ConvertedFunction(nil)
	if zero != nil {
		return 6
	}
	print("PASS\n")
	return 0
}
