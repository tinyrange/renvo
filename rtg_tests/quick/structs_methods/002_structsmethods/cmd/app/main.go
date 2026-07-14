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
	p := pair{a: 3, b: 8}
	p.add(3)
	if p.score() == 41 {
		print("PASS\n")
		return
	} else {

		print("FAIL\n")
	}
}
