package main

func quotient(v uint32, d uint32) uint32  { return v / d }
func remainder(v uint32, d uint32) uint32 { return v % d }

func appMain(args []string) int {
	if quotient(37, 10) != 3 || remainder(37, 10) != 7 {
		print("FAIL: small unsigned division\n")
		return 1
	}
	if quotient(4294967295, 10) != 429496729 || remainder(4294967295, 10) != 5 {
		print("FAIL: high bit unsigned division\n")
		return 1
	}
	if quotient(4294967295, 2147483648) != 1 || remainder(4294967295, 2147483648) != 2147483647 {
		print("FAIL: unsigned divisor\n")
		return 1
	}
	print("PASS\n")
	return 0
}
