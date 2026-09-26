package main

func main() {
	count := 0
	switch 1 {
	case 1:
		switch 1 {
		case 1:
			count++
		default:
			panic("inner default")
		}
	default:
		panic("outer default")
	}
	a, b := 1, 1
	switch 1 {
	case a:
		count++
	case b:
		panic("runtime-valued case order")
	}
	m := map[int]int{a: 3, b: 4}
	if len(m) != 1 || m[1] != 4 {
		panic("runtime-valued map keys")
	}
	var x interface{} = int8(1)
	switch x {
	case int(1):
		panic("distinct dynamic types")
	case int8(1):
		count++
	}
	switch x.(type) {
	case int:
		panic("type case")
	case int8:
		count++
	}
	nested := map[int]map[int]int{1: {1: 2}, 2: {1: 3}}
	if nested[1][1] != 2 || nested[2][1] != 3 || count != 4 {
		panic("independent cases and keys")
	}
	println("PASS")
}
