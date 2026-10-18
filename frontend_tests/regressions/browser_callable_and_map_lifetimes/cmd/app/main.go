package main

type Fields struct{ First, Second uint8 }
type Child struct{ Value int }

func (c *Child) Read(add int) int { return c.Value + add }

type Parent struct{ Child *Child }

func invoke(f func(int) int) int { return f(3) }
func callable(p *Parent, i Fields) int {
	var v [16]byte
	array := func(v [16]byte) (out [16]byte) { out = v; out[15]++; return out }
	v = array(v)
	first, second := i.First, i.Second
	closure := func() int { return int(first) + int(second) }
	first++
	bound := p.Child.Read
	return closure() + invoke(p.Child.Read) + bound(int(v[15]))
}
func main() {
	if callable(&Parent{Child: &Child{Value: 7}}, Fields{First: 2, Second: 4}) != 25 {
		panic("callable type/capture")
	}
	var nilMap map[int]*Child
	clear(nilMap)
	m := make(map[int]*Child)
	old := &Child{Value: 99}
	for round := 0; round < 80; round++ {
		for j := 0; j < 200; j++ {
			m[j] = old
		}
		if len(m) != 200 {
			panic("map fill")
		}
		visited := 0
		for range m {
			visited++
			clear(m)
		}
		if visited != 1 || len(m) != 0 || old.Value != 99 {
			panic("clear during iteration")
		}
		for j := 200; j < 600; j++ {
			m[j] = &Child{Value: round + j}
		}
		for j := 0; j < 200; j++ {
			if m[j] != nil {
				panic("stale clear entry")
			}
		}
		for j := 200; j < 600; j++ {
			if m[j].Value != round+j {
				panic("rehash after clear")
			}
		}
		clear(m)
	}
	println("PASS")
}
