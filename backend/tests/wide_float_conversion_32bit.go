package main

func appMain(args []string) int {
	value := complex(1.5, -2.5)
	if int64(real(value)) != 1 || int64(imag(value)) != -2 {
		print("FAIL: wide float conversion\n")
		return 1
	}
	print("PASS\n")
	return 0
}
