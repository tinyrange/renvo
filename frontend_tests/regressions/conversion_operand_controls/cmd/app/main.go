package main

type Byte byte
type Bytes []Byte
type Text string

func main() {
	bytes := []byte("abc")
	runes := []rune("a世")
	named := Bytes("xyz")
	if len(bytes) != 3 || bytes[1] != 'b' || len(runes) != 2 || runes[1] != '世' || named[2] != 'z' {
		panic("slice conversions")
	}
	if Text(bytes) != "abc" || string(65) != "A" || !bool(true) {
		panic("scalar conversions")
	}
	var value float64 = 3.5
	if int(value) != 3 || float64(2) != 2 {
		panic("numeric conversions")
	}
	int := func(s string) int { return len(s) }
	if int("abcd") != 4 {
		panic("shadowing")
	}
	println("PASS")
}
