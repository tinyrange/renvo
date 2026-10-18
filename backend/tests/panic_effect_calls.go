package main

var afterFault int

func pureCycleA(n int) int {
	if n == 0 {
		return 7
	}
	return pureCycleB(n - 1)
}
func pureCycleB(n int) int {
	if n == 0 {
		return 7
	}
	return pureCycleA(n - 1)
}
func pureForward(n int) int            { return pureCycleA(n) + 1 }
func faultIndex(v []int, i int) int    { return v[i] }
func faultPointer(p *int) int          { return *p }
func faultDivision(n, d int) int       { return n / d }
func faultAllocation(n int) []int      { return make([]int, n) }
func faultAssertion(v interface{}) int { return v.(int) }
func faultForward(kind int) {
	if kind == 0 {
		var v []int
		faultIndex(v, 1)
	}
	if kind == 1 {
		var p *int
		faultPointer(p)
	}
	if kind == 2 {
		faultDivision(7, 0)
	}
	if kind == 3 {
		faultAllocation(-1)
	}
	if kind == 4 {
		faultAssertion("wrong type")
	}
	afterFault++
}
func faultBoundary(kind int) (caught bool) {
	defer func() { caught = recover() != nil }()
	faultForward(kind)
	return
}
func stealPermission() bool { return recover() != nil }
func duringPanic() (caught, stolen bool, result int) {
	defer func() { caught = recover() != nil }()
	defer func() { result = pureForward(12); stolen = stealPermission() }()
	panic("owned")
}
func nestedFault() (outer, inner bool) {
	defer func() { outer = recover() != nil }()
	defer func() {
		defer func() { inner = recover() != nil }()
		var p *int
		faultPointer(p)
	}()
	panic("outer original")
}
func replacementFault() (caught interface{}) {
	defer func() { caught = recover() }()
	defer func() { var p *int; faultPointer(p) }()
	panic("replaced original")
}
func appMain() int {
	if pureForward(11) != 8 {
		panic("recursive pure effect")
	}
	for i := 0; i < 5; i++ {
		if !faultBoundary(i) {
			panic("lost implicit fault")
		}
	}
	if afterFault != 0 {
		panic("continued after fault")
	}
	caught, stolen, result := duringPanic()
	if !caught || stolen || result != 8 {
		panic("changed panic ownership")
	}
	outer, inner := nestedFault()
	if !outer || !inner {
		panic("lost nested panic chain")
	}
	if replacementFault().(int) != 1 {
		panic("wrong replacement fault value")
	}
	print("PASS\n")
	return 0
}
