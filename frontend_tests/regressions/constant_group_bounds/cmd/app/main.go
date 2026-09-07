package main

const (
	A, B = iota + 253, iota + 254
	C, D
)
const (
	_ = iota
	E
	F
)
const Reset = iota

var values [F]int

func main() {
	a := append([]byte{}, A, B, C, D)
	if a[0] != 253 || a[1] != 254 || a[2] != 254 || a[3] != 255 {
		panic("pairs")
	}
	if E != 1 || F != 2 || Reset != 0 || len(values) != 2 {
		panic("iota")
	}
	println("PASS")
}
