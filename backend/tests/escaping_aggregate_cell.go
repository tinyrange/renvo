package main

type cellBox struct{ value *int }

func makeCell(value int) cellBox {
	local := value
	return cellBox{value: &local}
}

func appMain(args []string) int {
	first := makeCell(10)
	second := makeCell(20)
	if *first.value != 10 || *second.value != 20 || first.value == second.value {
		panic("escaping aggregate cell")
	}
	*first.value += 3
	if *first.value != 13 || *second.value != 20 {
		panic("distinct cells")
	}
	print("PASS\n")
	return 0
}
