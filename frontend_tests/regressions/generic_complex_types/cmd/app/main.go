package main

type Small complex64
type Large complex128

func calculate[T ~complex64 | ~complex128](a, b T) T { return -a + b*2 - b/2 }
func increment[T ~complex64 | ~complex128](v T) T    { return v + 0.5i }
func boxed[T ~complex64 | ~complex128](v T) any      { return v + 0.5 }
func equal[T ~complex64 | ~complex128](v T) bool     { return v == 0.3-0.1i }
func verify[T ~complex64 | ~complex128]() {
	a, b := T(1+2i), T(4+6i)
	if calculate(a, b) != T(5+7i) {
		panic("arithmetic")
	}
	if increment(a) != T(1+2.5i) {
		panic("imaginary increment")
	}
	v, ok := boxed(a).(T)
	if !ok || v != T(1.5+2i) {
		panic("type identity")
	}
	if !equal(T(0.3 - 0.1i)) {
		panic("comparison")
	}
}
func main() {
	verify[complex64]()
	verify[complex128]()
	verify[Small]()
	verify[Large]()
	if Small(0.3-0.1i) != Small(complex(0.3, -0.1)) {
		panic("ordinary literal")
	}
	print("PASS\n")
}
