package main

type aliasCellOuter struct {
	box *aliasCellBox
}

type aliasCellBox struct {
	values [2]int
}

func aliasSetInt(p *int, value int) int {
	*p = value
	return value
}

func aliasCellCheck() *int {
	x := 1
	p := &x
	if aliasSetInt(p, 2) != 2 || x != 2 {
		panic("scalar alias read")
	}
	box := aliasCellBox{values: [2]int{3, 4}}
	q := &box.values[1]
	if aliasSetInt(q, 5) != 5 || box.values[1] != 5 {
		panic("aggregate alias read")
	}
	outer := aliasCellOuter{box: &box}
	outer.box.values[0]++
	if outer.box.values[0] != 4 {
		panic("field name aliases local")
	}
	x, *p = 6, 7
	if x != 7 || *p != 7 {
		panic("parallel alias write")
	}
	*p, x = 8, 9
	if x != 9 || *p != 9 {
		panic("reverse parallel alias write")
	}
	return p
}

func appMain(args []string) int {
	if *aliasCellCheck() != 9 {
		return 1
	}
	print("PASS\n")
	return 0
}
