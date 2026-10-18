package main

type Value int
type Builder struct{ Offset int }

func (b *Builder) Constant(x uint64) Value        { return Value(x) + Value(b.Offset) }
func (b *Builder) Binary(k int, a, c Value) Value { return a + c + Value(k) }
func (b *Builder) Domain() int                    { return b.Offset }

type Callbacks struct {
	Domain  func() int
	Address func(uint64) Value
}

func grouped(b *Builder) Value {
	constant, binary := b.Constant, b.Binary
	f := func(x Value) Value { return binary(3, x, constant(4)) }
	return f(10)
}
func main() {
	b := &Builder{Offset: 2}
	if grouped(b) != 19 {
		panic("grouped capture")
	}
	var c Callbacks
	c.Domain, c.Address = b.Domain, b.Constant
	if c.Domain() != 2 || c.Address(7) != 9 {
		panic("parallel method values")
	}
	best := []Value(nil)
	parts := []Value{42}
	f := func() { best = append([]Value(nil), parts...) }
	f()
	if len(best) != 1 || best[0] != 42 {
		panic("slice capture")
	}
	println("PASS")
}
