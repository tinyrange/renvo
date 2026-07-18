package main

import "example.com/rtgtests/regressions/interface_dynamic_dispatch/lib"

func verify(value lib.Scorer, score int, left int, right int, first int, second int) bool {
	if value.Score(3) != score {
		return false
	}
	pair := value.Pair()
	if pair.Left != left || pair.Right != right {
		return false
	}
	a, b := value.Split()
	return a == first && b == second
}

func main() {
	alpha := lib.Choose(true)
	beta := lib.Choose(false)
	alphaValue := lib.Alpha{Base: 30}
	var alphaPointer lib.Scorer = &alphaValue
	if !verify(alpha, 13, 10, 11, 10, 12) ||
		!verify(beta, 43, 40, 60, 80, 100) ||
		!verify(alphaPointer, 33, 30, 31, 30, 32) {
		print("FAIL\n")
		return
	}
	values := []lib.Scorer{alpha, beta, alphaPointer}
	if values[0].Score(1) != 11 || values[1].Score(1) != 41 || values[2].Score(1) != 31 {
		print("FAIL\n")
		return
	}
	var narrowed lib.Narrow = lib.NarrowFrom(beta)
	if narrowed.Score(4) != 44 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
