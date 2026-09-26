package main

func close(a, b, c, d float64) bool { return a == 0 && b == 0 && c == 1 && d == 0 }
func complex(a int) int             { return a + 10 }
func real(a int) int                { return a + 20 }
func imag(a int) int                { return a + 30 }
func recover() int                  { return 40 }
func main() {
	if !close(0., 0, 1, 0) || complex(1) != 11 || real(2) != 22 || imag(3) != 33 || recover() != 40 {
		panic("package builtin shadow")
	}
	println("PASS")
}
