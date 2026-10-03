package main

type pair struct{ x, y float64 }
type matrix struct{ a, b, c, d float64 }
type object struct{}

func (o *object) late(tag int, m matrix) pair { return pair{x: m.a, y: m.d} }
func appMain() int {
	var o object
	p := o.late(16, matrix{a: 1, d: 1})
	if int(p.x) != 1 || int(p.y) != 1 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
