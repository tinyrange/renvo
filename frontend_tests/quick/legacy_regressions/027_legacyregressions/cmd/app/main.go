package main

type issue27Pair struct {
	a int
	b int
}

func main() {
	p := issue27Pair{}
	p.a, p.b = 3, 4
	if p.a == 3 && p.b == 4 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
