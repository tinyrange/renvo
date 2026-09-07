package main

type Number int
type Alias = Number
type Left []Number
type Right []Alias
type Byte = byte
type Bytes []Byte

func main() {
	a := Left{1}
	b := Right{2, 3}
	a = append(a, b...)
	if len(a) != 3 || a[0] != 1 || a[2] != 3 {
		panic("named slices")
	}
	a = append(a, nil...)
	if len(a) != 3 {
		panic("nil")
	}
	bytes := Bytes{'a'}
	bytes = append(bytes, "bc"...)
	if string(bytes) != "abc" {
		panic("string")
	}
	runes := []rune{65}
	runes = append(runes, []int32{66}...)
	if len(runes) != 2 || runes[1] != 66 {
		panic("rune alias")
	}
	{
		a := []string{"x"}
		a = append(a, []string{"y"}...)
		if a[1] != "y" {
			panic("shadow")
		}
	}
	println("PASS")
}
