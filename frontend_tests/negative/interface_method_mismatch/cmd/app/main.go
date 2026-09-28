package main

type I interface{ M(int) }
type T int

func (T) M(string) {}
func main() {
	var _ I = T(0)
}
