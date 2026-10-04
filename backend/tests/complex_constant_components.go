package main

func parts() int { return real(complex(40, 3)) + imag(2i) }
func appMain(args []string) int {
	if parts() != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
