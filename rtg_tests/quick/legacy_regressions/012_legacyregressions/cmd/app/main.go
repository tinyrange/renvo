package main

type issue12Pair struct {
	a int
	b string
}

func main() {
	p := issue12Pair{a: 1, b: "x"}
	q := p
	q.a = 9
	if p.a == 1 && q.a == 9 && q.b == "x" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
