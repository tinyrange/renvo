package main

type namedInitializer163 func(int) int
type receiverInitializer163 struct{ N int }

func (p *receiverInitializer163) Read(n int) int { return p.N + n }
func appMain() int {
	p := &receiverInitializer163{N: 7}
	var f namedInitializer163 = p.Read
	p.N = 11
	if f(3) != 14 {
		panic("typed method initializer")
	}
	print("PASS\n")
	return 0
}
