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
	p := pair{a: 7, b: 3}
	p.add(3)
	if p.score() == 76 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
