package main

func complex64RoundTrip(value complex64) complex64 {
	return value
}

func appMain(args []string) int {
	values := [2]complex64{complex(float32(1), float32(2)), complex(float32(3), float32(4))}
	values[0] = complex64RoundTrip(values[1])
	if real(values[0]) != 3 || imag(values[0]) != 4 {
		print("FAIL: complex64 composite round trip\n")
		return 1
	}
	print("PASS\n")
	return 0
}
