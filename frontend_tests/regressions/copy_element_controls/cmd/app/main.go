package main

type N int
type Alias = N
type Left []N
type Right []Alias
type Byte = byte
type Bytes []Byte

func main() {
	a := make(Left, 3)
	b := Right{4, 5}
	if copy(a, b) != 2 || a[0] != 4 || a[1] != 5 || a[2] != 0 {
		panic("named elements")
	}
	dst := make(Bytes, 3)
	if copy(dst, "abc") != 3 || string(dst) != "abc" {
		panic("byte alias")
	}
	runes := make([]rune, 2)
	source := []int32{65, 66}
	if copy(runes, source) != 2 || runes[1] != 66 {
		panic("rune alias")
	}
	{
		a := []string{"x", "y"}
		if copy(a, []string{"z"}) != 1 || a[0] != "z" {
			panic("shadow")
		}
	}
	println("PASS")
}
