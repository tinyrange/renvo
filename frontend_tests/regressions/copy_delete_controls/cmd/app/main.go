package main

type Slice []int
type Map map[int]int

func main() {
	a := Slice{1, 2, 3}
	b := Slice{4, 5}
	if copy(a, b) != 2 || a[0] != 4 || a[1] != 5 || a[2] != 3 {
		panic("slice copy")
	}
	bytes := make([]byte, 3)
	if copy(bytes, "abc") != 3 || string(bytes) != "abc" {
		panic("string copy")
	}
	m := Map{1: 2, 3: 4}
	delete(m, 1)
	delete(m, 9)
	if len(m) != 1 || m[3] != 4 {
		panic("delete")
	}
	var empty Map
	delete(empty, 1)
	value := 2
	{
		value := []int{1}
		if copy(value, []int{3}) != 1 || value[0] != 3 {
			panic("scope")
		}
	}
	copy := func(a, b int) int { return a + b }
	if copy(value, 3) != 5 {
		panic("shadowed copy")
	}
	println("PASS")
}
