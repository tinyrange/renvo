package main

type pair struct {
	a int
	b int
}

func (p pair) score() int {
	return p.a*10 + p.b
}

func (p *pair) add(v int) {
	p.b = p.b + v
}

func main() {
	p := pair{a: 10, b: 11}
	p.add(3)
	if p.score() == 114 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
