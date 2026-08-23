package main

type complex64UntypedHolder struct {
	value complex64
}

func complex64UntypedReturn() complex64 {
	return complex(3.5, 4.5)
}

func complex64UntypedAccept(value complex64) bool {
	return real(value) == 5.5 && imag(value) == 6.5
}

func appMain(args []string) int {
	var value complex64 = complex(1.5, 2.5)
	if real(value) != 1.5 || imag(value) != 2.5 {
		print("FAIL: complex64 untyped local\n")
		return 1
	}
	returned := complex64UntypedReturn()
	if real(returned) != 3.5 || imag(returned) != 4.5 || !complex64UntypedAccept(complex(5.5, 6.5)) {
		print("FAIL: complex64 untyped call context\n")
		return 1
	}
	var imaginary complex64 = 7.5 + 8.5i
	holder := complex64UntypedHolder{value: complex(11.5, 12.5)}
	if real(imaginary) != 7.5 || imag(imaginary) != 8.5 {
		print("FAIL: complex64 untyped imaginary context\n")
		return 1
	}
	if real(holder.value) != 11.5 || imag(holder.value) != 12.5 {
		print("FAIL: complex64 untyped field context\n")
		return 1
	}
	print("PASS\n")
	return 0
}
