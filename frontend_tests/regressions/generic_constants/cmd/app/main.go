package main

type namedFloat float64
type namedComplex complex64

func sum[T ~float32 | ~float64]() T { return 0.1 + 0.2 }
func identity[T any](v T) T         { return v }
func take(v float64) float64        { return v }
func rounded[T ~float32]() T        { return 1 + 0x1p-24 + 0x1p-100 }
func contexts[T ~float64]() {
	const fraction = 0.1 + 0.2
	var value T = fraction
	if value != 0.3 {
		panic("variable")
	}
	value = 0.1 + 0.2
	if value != 0.3 {
		panic("assignment")
	}
	if take(0.1+0.2) != 0.3 {
		panic("call")
	}
	if identity[float64](0.1+0.2) != 0.3 {
		panic("generic call")
	}
	if float64(0.1+0.2) != 0.3 {
		panic("conversion")
	}
	s := []T{0.1 + 0.2}
	s = append(s, 0.1+0.2)
	if s[0] != 0.3 || s[1] != 0.3 {
		panic("slice")
	}
	b := struct{ Value T }{0.1 + 0.2}
	if b.Value != 0.3 {
		panic("struct")
	}
	m := map[float64]T{0.1 + 0.2: 0.1 + 0.2}
	if m[0.1+0.2] != 0.3 {
		panic("map")
	}
	delete(m, 0.1+0.2)
	if len(m) != 0 {
		panic("delete")
	}
	switch value {
	case 0.1 + 0.2:
	default:
		panic("switch")
	}
	var boxed any = 0.1 + 0.2
	if boxed.(float64) != 0.3 {
		panic("boxing")
	}
	var f32 float32 = 0.1 + 0.2
	if float64(f32) != float64(float32(0.3)) {
		panic("typed rounding")
	}
	if max(f32, 0.1+0.2) != float32(0.3) {
		panic("max")
	}
	if complex(f32, 0.1+0.2) != complex64(0.3+0.3i) {
		panic("complex builtin")
	}
}
func complexSum[T ~complex64 | ~complex128]() T { return (0.1 + 0.2) + (0.2-0.3)*1i }
func main() {
	if sum[float64]() != 0.3 || sum[float32]() != float32(0.3) {
		panic("return")
	}
	if rounded[float32]() != 1+0x1p-23 {
		panic("double rounding")
	}
	c64, c128 := complexSum[complex64](), complexSum[complex128]()
	if real(c64) != float32(0.3) || imag(c64) != float32(-0.1) || real(c128) != 0.3 || imag(c128) != -0.1 {
		panic("complex return")
	}
	named := complexSum[namedComplex]()
	if real(named) != float32(0.3) || imag(named) != float32(-0.1) {
		panic("named complex")
	}
	contexts[float64]()
	contexts[namedFloat]()
	print("PASS\n")
}
