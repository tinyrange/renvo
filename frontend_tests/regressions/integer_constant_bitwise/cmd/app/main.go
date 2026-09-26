package main

var values [(16 | 3) & 3]int

func main() {
	a := append([]byte{}, 0x1234&255, 128|127, 255^128, 255&^128)
	if a[0] != 52 || a[1] != 255 || a[2] != 127 || a[3] != 127 {
		panic("bits")
	}
	b := append([]int8{}, -1&^127, -128|127, -1^127)
	if b[0] != -128 || b[1] != -1 || b[2] != -128 {
		panic("signed")
	}
	if len(values) != 3 {
		panic("array")
	}
	println("PASS")
}
