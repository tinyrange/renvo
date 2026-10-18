package main

type homogeneous163 struct{ N uint64 }
type homogeneousCall163 func(uint64, uint64) uint64
type homogeneousAlias163 func(uint64, uint64) uint64

func (p *homogeneous163) Sum(a, b uint64) uint64 { return p.N + a + b }
func (p *homogeneous163) Xor(a, b uint64) uint64 { return p.N ^ a ^ b }
func homogeneousPlain163(a, b uint64) uint64     { return a + b }
func homogeneousClosure163(base int32) func(int32, int32) int32 {
	return func(a, b int32) int32 { base += a; return base + b }
}
func environmentFreeA163(n int16) int16 { return n + 3 }
func environmentFreeB163(n int16) int16 { return n ^ 7 }
func homogeneousCheck163() {
	p := &homogeneous163{N: 9}
	var f homogeneousCall163 = p.Sum
	var g homogeneousAlias163 = p.Xor
	var h homogeneousCall163 = homogeneousPlain163
	p.N = 17
	if f(3, 5) != 25 || g(3, 5) != 23 || h(3, 5) != 8 || homogeneousAlias163(f)(3, 5) != 25 {
		panic("homogeneous methods")
	}
	one := homogeneousClosure163(3)
	two := homogeneousClosure163(7)
	same := one
	if one(2, 1) != 6 || two(3, 2) != 12 || same(4, 5) != 14 {
		panic("homogeneous closures")
	}
	free := []func(int16) int16{environmentFreeA163, environmentFreeB163}
	if free[0](9) != 12 || free[1](9) != 14 {
		panic("environment free")
	}
	var recursive func(int64) int64
	recursive = func(n int64) int64 {
		if n == 0 {
			return 0
		}
		return recursive(n-1) + n
	}
	if recursive(7) != 28 {
		panic("recursive environment")
	}
}

type env160 struct{ N int }

func (p *env160) Read(n int) int {
	if p == nil {
		return -n
	}
	return p.N + n
}
func (p *env160) Add(n int) int  { p.N += n; return p.N }
func (p env160) Value(n int) int { return p.N + n }

type face160 interface{ Read(int) int }
type callback160 func(int) int
type holder160 struct {
	F callback160
	G func(int) int
}

var saved160 []callback160

func named160(n int) int        { return n + 101 }
func make160(x int) callback160 { return func(n int) int { x += n; return x } }
func nilCall160() (ok bool) {
	defer func() { ok = recover() != nil }()
	var f callback160
	f(3)
	return false
}
func nilFace160() (ok bool) {
	defer func() { ok = recover() != nil }()
	var face face160
	_ = face.Read
	return false
}
func compare160(f callback160) (ok bool) {
	defer func() { ok = recover() != nil }()
	var a interface{} = f
	var b interface{} = f
	_ = a == b
	return false
}
func deferred160() (out int) {
	p := &env160{N: 17}
	f := p.Read
	defer func(g callback160) { out = g(3) }(f)
	f = named160
	p.N = 23
	return
}
func main() {
	f := make160(3)
	g := make160(7)
	copyF := f
	if f(2) != 5 || g(3) != 10 || copyF(4) != 9 {
		panic("closure identity")
	}
	p := &env160{N: 11}
	var face face160 = p
	ptr := p.Add
	snapshot := p.Value
	boundFace := face.Read
	p.N = 19
	if ptr(4) != 23 || snapshot(2) != 13 || boundFace(5) != 28 {
		panic("receiver snapshot")
	}
	var nilPointer *env160
	nilBound := nilPointer.Read
	if nilBound == nil || nilBound(7) != -7 || !nilCall160() || !nilFace160() {
		panic("nil")
	}
	h := holder160{F: f, G: g}
	m := map[int]holder160{1: h}
	hs := []holder160{m[1], h}
	if hs[0].F(1) != 10 || m[1].G(2) != 12 || hs[1].G(3) != 15 {
		panic("containers")
	}
	for i := 0; i < 80; i++ {
		saved160 = append(saved160, make160(i))
	}
	for i := 0; i < 80; i++ {
		if saved160[i](2) != i+2 {
			panic("escaped environment")
		}
	}
	if !compare160(f) || !compare160(named160) || deferred160() != 26 {
		panic("compare/defer")
	}
	homogeneousCheck163()
	print("PASS\n")
}
