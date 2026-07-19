package main

func floatLit18() float64 { return 10.5 }
func appMain(args []string) int {
	if floatLit18() != 10.5 {
		print("float_literals_18 return\n")
		return 1
	}
	print("PASS\n")
	return 0
}
