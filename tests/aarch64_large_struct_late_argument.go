package main

type largeArgPoint struct {
	x float64
	y float64
}

type largeArgMatrix struct {
	a  float64
	b  float64
	c  float64
	d  float64
	tx float64
	ty float64
}

type largeArgPath struct{}

func (m largeArgMatrix) transform(p largeArgPoint) largeArgPoint {
	return largeArgPoint{x: m.a*p.x + m.c*p.y + m.tx, y: m.b*p.x + m.d*p.y + m.ty}
}

func (p *largeArgPath) apply(index int, m largeArgMatrix) largeArgPoint {
	return m.transform(largeArgPoint{x: 56, y: 48})
}

func appMain(args []string) int {
	var p largeArgPath
	got := p.apply(16, largeArgMatrix{a: 1, d: 1})
	if int(got.x) == 56 && int(got.y) == 48 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
