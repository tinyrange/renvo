package main

func floatLit24a(x float64) float64 { return floatLit24b(x) + 1.0 }
func floatLit24b(x float64) float64 { return x * 2.0 }
func appMain(args []string) int {
	if floatLit24a(3.0) != 7.0 {
		print("float_literals_24 nested\n")
		return 1
	}
	print("PASS\n")
	return 0
}
