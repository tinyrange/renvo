package main

type Float float32
type Complex complex64

func main() {
	z := complex(3, 4)
	if real(z) != 3 || imag(z) != 4 {
		panic("components")
	}
	var a Float = 5
	var b Float = 6
	w := complex(a, b)
	if real(w) != 5 || imag(w) != 6 {
		panic("named float")
	}
	var named Complex = 7 + 8i
	if real(named) != 7 || imag(named) != 8 {
		panic("named complex")
	}
	if imag(2) != 0 || real(2.0) != 2 {
		panic("constant conversion")
	}
	{
		imag := func(s string) int { return len(s) }
		if imag("abc") != 3 {
			panic("shadow")
		}
	}
	println("PASS")
}
