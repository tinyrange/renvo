package main

type Value interface{ Set(int) }
type Item struct{ Value Value }
type Number struct{ n int }

func (n *Number) Set(value int) { n.n = value }

func main() {
	n := &Number{}
	item := Item{Value: n}
	item.Value.Set(3)
	if n.n != 3 {
		panic("interface field method")
	}
	setter := Value.Set
	setter(n, 4)
	if n.n != 4 {
		panic("interface method expression")
	}
	print("PASS\n")
}
