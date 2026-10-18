package main

type getterChild struct {
	N int
	W uint64
}
type getterNode struct {
	N     int
	B     byte
	S     int16
	U     uint32
	W     uint64
	F     float64
	Text  string
	Items []int
	Any   interface{}
	A     [5]uint64
	Child getterChild
	Next  *getterNode
}

const getterViewEnd = 4

func (p *getterNode) number() int              { return p.N }
func (p *getterNode) small() int16             { return p.S }
func (p *getterNode) wide() uint64             { return p.W }
func (p *getterNode) floating() float64        { return p.F }
func (p *getterNode) text() string             { return p.Text }
func (p *getterNode) items() []int             { return p.Items }
func (p *getterNode) boxed() interface{}       { return p.Any }
func (p *getterNode) child() getterChild       { return p.Child }
func (p *getterNode) array() [5]uint64         { return p.A }
func (p *getterNode) address() *int            { return &p.Child.N }
func (p *getterNode) through() int             { return p.Next.Child.N }
func (p *getterNode) nextAddress() *uint64     { return &p.Next.W }
func (p *getterNode) view() []uint64           { return p.A[:] }
func (p *getterNode) boundedView() []uint64    { return p.A[1:getterViewEnd:5] }
func (p *getterNode) emptyView() []uint64      { return p.A[0:0] }
func getterPlain(p *getterNode) uint32         { return p.U }
func getterIdentity(p *getterNode) *getterNode { return p }

var getterEffects int

func getterReceiver(p *getterNode) *getterNode { getterEffects++; return p }
func getterNotPure(p *getterNode) int          { getterEffects++; return p.N }
func getterNotPath(p *getterNode) int          { return getterNotPure(p) }

type getterInterface interface {
	number() int
	view() []uint64
	boxed() interface{}
}
type getterOther struct {
	Value int
	A     [2]uint64
}

func (p *getterOther) number() int        { return p.Value }
func (p *getterOther) view() []uint64     { return p.A[:] }
func (p *getterOther) boxed() interface{} { return p.Value }
func getterDynamic(i getterInterface, n int) int {
	if i.number() != n || len(i.view()) == 0 || i.boxed() == nil {
		panic("dynamic getter")
	}
	f := i.number
	return f()
}
func getterIndirectRecover() bool { return recover() != nil }
func getterPanicOwner(p *getterNode, direct *bool, stolen *bool) {
	if p.number() != 11 || len(p.view()) != 5 || p.boxed() != 33 {
		panic("owner getters")
	}
	*stolen = getterIndirectRecover()
	*direct = recover() != nil
}
func getterOwnerCase(p *getterNode) (direct bool, stolen bool) {
	defer getterPanicOwner(p, &direct, &stolen)
	panic("outer")
}
func getterFault(kind int) (caught bool) {
	defer func() { caught = recover() != nil }()
	var p *getterNode
	if kind >= 8 {
		p = &getterNode{}
	}
	if kind == 0 {
		_ = p.number()
	}
	if kind == 1 {
		_ = p.address()
	}
	if kind == 2 {
		_ = p.view()
	}
	if kind == 3 {
		_ = p.emptyView()
	}
	if kind == 4 {
		_ = p.boxed()
	}
	if kind == 5 {
		_ = p.wide()
	}
	if kind == 6 {
		_ = p.array()
	}
	if kind == 7 {
		var i getterInterface = p
		_ = i.number()
	}
	if kind == 8 {
		_ = p.through()
	}
	if kind == 9 {
		_ = p.nextAddress()
	}
	if kind == 10 {
		defer p.Next.number()
	}
	return
}
func appMain() int {
	q := getterNode{N: 19, W: 0x8877665544332211, Child: getterChild{N: 23}}
	p := getterNode{N: 11, B: 251, S: -1234, U: 0xf1234567, W: 0xfedcba9876543210, F: -2.5,
		Text: "getter", Items: []int{3, 7}, Any: 33, A: [5]uint64{1, 2, 3, 4, 5}, Child: getterChild{N: 13, W: 17}, Next: &q}
	if getterReceiver(&p).number() != 11 || getterEffects != 1 {
		panic("receiver once")
	}
	if p.small() != -1234 || getterPlain(&p) != 0xf1234567 || p.wide() != 0xfedcba9876543210 || p.floating() != -2.5 {
		panic("numeric getters")
	}
	if p.text() != "getter" || p.boxed() != 33 || p.child().W != 17 || p.array()[4] != 5 {
		panic("value getters")
	}
	items := p.items()
	items[0] = 29
	*p.address() = 31
	v := p.view()
	v[2] = 41
	b := p.boundedView()
	if p.Items[0] != 29 || p.Child.N != 31 || p.A[2] != 41 || len(b) != 3 || cap(b) != 4 || b[1] != 41 || len(p.emptyView()) != 0 {
		panic("alias and bounds")
	}
	*p.nextAddress() = 43
	if p.through() != 23 || q.W != 43 {
		panic("pointer chain")
	}
	var nilNode *getterNode
	if getterIdentity(nilNode) != nil {
		panic("nil identity")
	}
	if getterNotPath(&p) != 11 || getterEffects != 2 {
		panic("non getter effects")
	}
	if getterDynamic(&p, 11) != 11 || getterDynamic(&getterOther{Value: 47, A: [2]uint64{1, 2}}, 47) != 47 {
		panic("polymorphic dispatch")
	}
	method := p.wide
	plain := getterPlain
	if method() != p.W || plain(&p) != p.U {
		panic("function values")
	}
	// Callee lexical names must not bind to caller locals or closure cells.
	getterViewEnd := 2
	f := func() int { return getterViewEnd }
	if len(p.boundedView()) != 3 || f() != 2 {
		panic("lexical constants")
	}
	for i := 0; i < 11; i++ {
		if !getterFault(i) {
			panic("nil getter did not fault")
		}
	}
	direct, stolen := getterOwnerCase(&p)
	if !direct || stolen {
		panic("recover ownership")
	}
	print("PASS\n")
	return 0
}
