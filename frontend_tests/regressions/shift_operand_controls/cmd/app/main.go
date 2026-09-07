package main

type Count uint

func main() {
	const a = 2.0 << 1
	const b = 1 << 2.0
	const c = 1<<2 + 0.5
	if a != 4 || b != 4 || c != 4.5 {
		panic("constant shift")
	}
	var count Count = 3
	if 1<<count != 8 || 1<<2<<3 != 32 || 2*3<<1 != 12 {
		panic("precedence")
	}
	n := 2.5
	{
		n := 2
		if 1<<n != 4 {
			panic("shadowing")
		}
	}
	if n != 2.5 {
		panic("outer binding")
	}
	println("PASS")
}
