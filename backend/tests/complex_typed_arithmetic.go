package main

type Small complex64
type Large complex128

func small(a Small, b Small) Small { return a + b*2 - b/2 }
func large(a Large, b Large) Large { return a + b*2 - b/2 }
func appMain(args []string) int {
	a, b := Small(complex(1, 2)), Small(complex(4, 6))
	c, d := Large(complex(1, 2)), Large(complex(4, 6))
	x, y := small(a, b), large(c, d)
	if real(x) != 7 || imag(x) != 11 || real(y) != 7 || imag(y) != 11 {
		return 1
	}
	var boxed interface{} = a + 0.5
	z, ok := boxed.(Small)
	if !ok || real(z) != 1.5 || imag(z) != 2 {
		return 2
	}
	print("PASS\n")
	return 0
}
