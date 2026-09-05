package main

func main() {
	const (
		A    = iota
		iota = iota
		B
		C
	)
	if A != 0 || B != 1 || C != 1 {
		panic("implicit constant scope")
	}
	println("PASS")
}
