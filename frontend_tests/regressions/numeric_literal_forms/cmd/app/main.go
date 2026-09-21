package main

func main() {
	a := 0b_1010
	b := 0o_17
	c := 0x_ff
	d := 0_123
	if a != 10 || b != 15 || c != 255 || d != 83 {
		panic("integers")
	}
	const x = 0x_1.8p+1
	const y = 1_2.5e-1
	if x != 3 || y != 1.25 {
		panic("floats")
	}
	println("PASS")
}
