package main

const size = 1

func main() {
	const size = size + 1
	const saved = size
	const (
		_ = iota
		index
	)
	a := [size]int{index: 7}
	b := [...]int{index: 3, 4}
	var declared [size]int
	var left, right [size]int
	matrix := [size][size]int{{1, 2}, {3, 4}}
	pointer := &[size]int{5, 6}
	if len(declared) != 2 || len(left) != 2 || len(right) != 2 || matrix[1][1] != 4 || pointer[1] != 6 {
		panic("nested and declared arrays")
	}
	{
		const size = 0
		c := [saved]int{1, 2}
		if size != 0 || len(c) != 2 || c[1] != 2 {
			panic("declaration scope")
		}
	}
	if len(a) != 2 || a[0] != 0 || a[1] != 7 || len(b) != 3 || b[1] != 3 || b[2] != 4 {
		panic("local array constants")
	}
	println("PASS")
}
