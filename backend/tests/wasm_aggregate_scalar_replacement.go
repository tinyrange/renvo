package main

type Pair158 struct{ A, B int }
type Inner158 struct {
	A int
	B uint64
}
type Record158 struct {
	Tag  int
	In   Inner158
	Last int
}
type Narrow158 struct {
	A byte
	B uint16
	C int32
	D uint64
}
type Embed158 struct {
	*Pair158
	Z int
}

var effects158 int

func rhs158(x int) int        { effects158++; return x }
func wide158(x uint64) uint64 { effects158++; return x }
func pass158(p Record158, n int) Record158 {
	if n == 0 {
		return p
	}
	p.In.A = p.In.A + n
	p.In.B = p.In.B ^ uint64(n)
	return pass158(p, n-1)
}
func sum158(p Record158) uint64   { return uint64(p.Tag+p.In.A+p.Last) + p.In.B }
func (p Record158) Score() uint64 { return sum158(p) }
func scalar158(p Pair158) int {
	var q Pair158
	q = p
	q.A = rhs158(q.B + 3)
	q.B = rhs158(q.A + q.B)
	q = q
	return q.A + q.B
}
func touch158(p *int) { *p = *p + 19 }
func alias158(p Record158) int {
	q := p
	x := &q.In.A
	q.Tag = rhs158(31)
	touch158(x)
	q.In.B = wide158(0x8899aabbccddeeff)
	return q.In.A + q.Tag + q.Last
}
func capture158() int {
	q := Pair158{A: 3, B: 7}
	f := func() int { q.A = q.A + q.B; return q.A }
	q.B = 11
	a := f()
	q.A = rhs158(29)
	return a + f()
}
func fault158() (ok bool) {
	q := Pair158{A: 5, B: 9}
	defer func() {
		if recover() != nil {
			ok = q.A == 5 && q.B == 9
		}
	}()
	var p *Pair158
	q.A = rhs158(p.B)
	return false
}
func faultWide158() (ok bool) {
	q := Record158{Tag: 3, In: Inner158{A: 7, B: 0xfedcba9876543210}, Last: 11}
	defer func() {
		if recover() != nil {
			ok = q.In.B == 0xfedcba9876543210 && q.In.A == 7
		}
	}()
	var p *Inner158
	q.In.B = wide158(p.B)
	return false
}
func lifetimes158() int {
	total := 0
	{
		p := Record158{Tag: 1, In: Inner158{A: 2, B: 0x1122334455667788}, Last: 3}
		x := &p.In.A
		*x = 17
		total = p.Tag + p.In.A + p.Last
	}
	{
		q := Pair158{A: 23, B: 29}
		q.A = rhs158(q.A + q.B)
		total = total + q.A + q.B
	}
	{
		p := Record158{Tag: 31, In: Inner158{A: 37, B: 0x9988776655443322}, Last: 41}
		if p.In.B != 0x9988776655443322 {
			return -1
		}
		total = total + p.Tag + p.In.A + p.Last
	}
	return total
}
func external158(src []Record158, dst []Record158) uint64 {
	q := src[1]
	q.In.A = rhs158(q.In.A + 17)
	q.In.B = wide158(q.In.B ^ 0x123456789abcdef0)
	dst[0] = q
	return sum158(q)
}

func appMain() int {
	effects158 = 0
	if scalar158(Pair158{A: 2, B: 7}) != 27 || effects158 != 2 {
		panic("scalar")
	}
	p := Record158{Tag: 2, In: Inner158{A: 3, B: 0xfedcba9876543210}, Last: 5}
	q := p
	q.In.A = rhs158(17)
	q.In.B = wide158(0x1122334455667788)
	if p.In.A != 3 || p.In.B != 0xfedcba9876543210 || q.In.A != 17 || q.In.B != 0x1122334455667788 {
		panic("copy")
	}
	q.In.B = q.In.B ^ 0x123456789abcdef0
	if q.In.B != (uint64(0x1122334455667788) ^ uint64(0x123456789abcdef0)) {
		panic("wide")
	}
	f := sum158
	if f(p) != 0xfedcba987654321a {
		panic("value call")
	}
	method := p.Score
	if method() != f(p) {
		panic("bound")
	}
	r := pass158(p, 3)
	if r.In.A != 9 || r.In.B != p.In.B || p.In.A != 3 {
		panic("recursion")
	}
	if alias158(p) != 58 {
		panic("alias")
	}
	if capture158() != 54 {
		panic("capture")
	}
	before := effects158
	if !fault158() || !faultWide158() || effects158 != before {
		panic("fault")
	}
	if lifetimes158() != 211 {
		panic("lifetimes")
	}
	narrow := Narrow158{A: 0xfe, B: 0xabcd, C: -37, D: 0x9876543210abcdef}
	narrow.B = 0xface
	narrow.C = -71
	if narrow.A != 0xfe || narrow.B != 0xface || narrow.C != -71 || narrow.D != 0x9876543210abcdef {
		panic("lanes")
	}
	base := Pair158{A: 7, B: 11}
	embedded := Embed158{Pair158: &base, Z: 13}
	embedded.A = 19
	if base.A != 19 || embedded.B != 11 || embedded.Z != 13 {
		panic("embedded pointer")
	}
	a := [3]Record158{p, q, r}
	a[1] = a[0]
	if a[1].In.B != p.In.B || a[2].In.A != 9 {
		panic("array copy")
	}
	// External loads/stores must synchronize every lane, even when the two
	// slice views overlap and the outgoing copy replaces the original source.
	external := []Record158{p, r, p}
	want := r
	want.In.A = want.In.A + 17
	want.In.B = want.In.B ^ 0x123456789abcdef0
	before = effects158
	if external158(external, external[1:]) != sum158(want) || effects158 != before+2 || external[1] != want || external[0] != p {
		panic("external overlap")
	}
	if external158(external, external[:1]) == 0 || external[1] != want {
		panic("external reverse overlap")
	}
	print("PASS\n")
	return 0
}
