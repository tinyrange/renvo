package main

func le(a, b float64) bool        { return a <= b }
func ge(a, b float64) bool        { return a >= b }
func leExpr(a, b, c float64) bool { return a-b <= c }

func appMain(args []string) int {
	if !le(3, 3) {
		print("FAIL\n")
		return 1
	}
	if !ge(3, 3) {
		print("FAIL\n")
		return 1
	}
	if le(2, 3) == false {
		print("FAIL\n")
		return 1
	}
	if le(3, 2) {
		print("FAIL\n")
		return 1
	}
	if !ge(4, 3) {
		print("FAIL\n")
		return 1
	}
	if !leExpr(1, 1, 0) {
		print("FAIL\n")
		return 1
	}
	x := le(2, 2)
	if !x {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
