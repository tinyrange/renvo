package main

type I interface{ M() }
type T int

func (*T) M() {}
func main() {
	var _ I = T(0)
}
