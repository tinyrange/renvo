package main

const Dividend = 257

var values [Dividend / 128]int

func main() {
	a := append([]byte{}, Dividend/2, Dividend%128)
	b := append([]int8{}, -Dividend/2, -Dividend%129)
	if len(values) != 2 || a[0] != 128 || a[1] != 1 {
		panic("positive")
	}
	if b[0] != -128 || b[1] != -128 {
		panic("negative")
	}
	c := append([]int{}, Dividend/-128, -Dividend/-128)
	if c[0] != -2 || c[1] != 2 {
		panic("sign")
	}
	println("PASS")
}
