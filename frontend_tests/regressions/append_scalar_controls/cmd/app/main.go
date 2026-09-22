package main

type Number int
type Alias = Number
type Numbers []Number
type Text string

func main() {
	var n Alias = 3
	a := append(Numbers{1}, 2.0, n)
	if len(a) != 3 || a[1] != 2 || a[2] != 3 {
		panic("named")
	}
	b := append([]byte{}, uint8(65), 'B')
	if string(b) != "AB" {
		panic("byte")
	}
	c := append([]rune{}, int32(67), 'D')
	if c[0] != 67 || c[1] != 68 {
		panic("rune")
	}
	s := append([]Text{}, "hello", Text("world"))
	if s[0] != "hello" || s[1] != "world" {
		panic("string")
	}
	flag := true
	flags := append([]bool{}, false, flag)
	if flags[0] || !flags[1] {
		panic("bool")
	}
	{
		a := []string{}
		a = append(a, "shadow")
		if a[0] != "shadow" {
			panic("scope")
		}
	}
	println("PASS")
}
