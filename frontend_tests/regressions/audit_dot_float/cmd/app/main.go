package main

func main() {
	x := .5
	y := +.01
	if x != 0.5 || y != 0.01 || .5e2 != 50 || -.25 != -0.25 { panic("leading dot") }
	println("PASS")
}
