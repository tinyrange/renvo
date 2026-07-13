package main

type issue39I interface{ V() int }
type issue39P struct{ n int }

func (p issue39P) V() int { return p.n }

func main() {
	p := issue39P{n: 1}
	var i issue39I = p
	p.n = 9
	if i.V() == 1 && p.n == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
