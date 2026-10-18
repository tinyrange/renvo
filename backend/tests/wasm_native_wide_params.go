package main

type paramU uint64
type paramS int64
type paramPair struct{ lo, hi uint64 }
type paramFace interface{ Mix(uint64, int) uint64 }
type paramObj struct{ v uint64 }

func (p *paramObj) Mix(x uint64, n int) uint64               { return (p.v ^ x) + uint64(n) }
func param0(a, b, c, d int, x paramU) paramU                 { return x + paramU(a+b+c+d) }
func param1(a, b, c int, x paramU, d int) paramU             { return x + paramU(a+b+c+d) }
func param2(a, b int, x paramU, c, d int) paramU             { return x + paramU(a+b+c+d) }
func param3(a int, x paramU, b, c, d int) paramU             { return x + paramU(a+b+c+d) }
func param4(x paramU, a, b, c, d int) paramU                 { return x + paramU(a+b+c+d) }
func paramMany(a, b, c, d, e, f int, x paramU, g int) paramU { return x + paramU(a+b+c+d+e+f+g) }
func paramRecursive(n int, x paramU) paramU {
	if n == 0 {
		return x
	}
	return paramRecursive(n-1, x+paramU(n))
}
func paramTwo(x paramU, y paramS) paramPair { return paramPair{uint64(x), uint64(y)} }
func paramFloat(x float64, i int) float64   { return x + float64(i) }
func paramRecover(x uint64) (result uint64) {
	defer func() {
		if recover() != nil {
			result = x
		}
	}()
	panic(x)
}
func paramCaller(f func(int, paramU) paramU, x paramU) paramU { return f(4, x) }
func appMain() int {
	x := paramU(0xfedcba9887654321)
	if param0(1, 2, 3, 4, x) != x+10 || param1(1, 2, 3, x, 4) != x+10 || param2(1, 2, x, 3, 4) != x+10 || param3(1, x, 2, 3, 4) != x+10 || param4(x, 1, 2, 3, 4) != x+10 {
		return 1
	}
	if paramMany(1, 2, 3, 4, 5, 6, x, 7) != x+28 || paramRecursive(7, x) != x+28 {
		return 2
	}
	p := paramTwo(x, paramS(-0x123456781234567))
	if p.lo != uint64(x) || p.hi != uint64(paramS(-0x123456781234567)) {
		return 3
	}
	if paramFloat(-2.25, 4) != 1.75 || paramFloat(1.0, -3) != -2.0 {
		return 4
	}
	o := &paramObj{v: uint64(x)}
	var face paramFace = o
	if face.Mix(0x123456789abcdef0, 5) != (uint64(x)^0x123456789abcdef0)+5 {
		return 5
	}
	bound := o.Mix
	if bound(0x123456789abcdef0, 5) != (uint64(x)^0x123456789abcdef0)+5 {
		return 6
	}
	if paramCaller(paramRecursive, x) != x+10 || paramRecover(uint64(x)) != uint64(x) {
		return 7
	}
	captured := func(i int, y paramU) paramU { return x + y + paramU(i) }
	if paramCaller(captured, 3) != x+7 {
		return 8
	}
	print("PASS\n")
	return 0
}
