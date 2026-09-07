package main

func sum(v ...int) int {
	result := 0
	for _, n := range v {
		result += n
	}
	return result
}

func main() {
	const a = .5
	const b = 1.
	const c = 0x1.8p1
	if a+b != 1.5 || c != 3 {
		panic("numbers")
	}
	x := struct{ Field int }{Field: 7}
	var y interface{} = x.Field
	if y.(int) != 7 {
		panic("selector/assertion")
	}
	values := [...]int{1, 2}
	if sum(values[:]...) != 3 {
		panic("ellipsis")
	}
	println("PASS")
}
