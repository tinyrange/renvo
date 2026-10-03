package main

type Value complex64

func value() Value { return Value(0.5 - 0.25i) }
func appMain(args []string) int {
	v := value()
	if real(v) != 0.5 || imag(v) != -0.25 {
		return 1
	}
	print("PASS\n")
	return 0
}
