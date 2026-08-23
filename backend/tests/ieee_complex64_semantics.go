package main

type ieeeComplex64Container struct {
	before byte
	value  complex64
	after  byte
}

func appMain(args []string) int {
	left := complex(float32(2), float32(3))
	right := complex(float32(4), float32(5))
	product := left * right
	if real(product) != -7 || imag(product) != 22 {
		print("FAIL: complex64 arithmetic\n")
		return 1
	}

	values := [2]complex64{left, right}
	container := ieeeComplex64Container{before: 7, value: values[1], after: 9}
	if container.before != 7 || real(container.value) != 4 || imag(container.value) != 5 || container.after != 9 {
		print("FAIL: complex64 layout\n")
		return 1
	}

	var boxed interface{} = product
	unboxed, ok := boxed.(complex64)
	if !ok || real(unboxed) != -7 || imag(unboxed) != 22 {
		print("FAIL: complex64 interface\n")
		return 1
	}
	print("PASS\n")
	return 0
}
