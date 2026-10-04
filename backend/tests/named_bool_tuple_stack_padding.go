package main

func poisonBoolTupleFrame() {
	var words [128]int
	for i := 0; i < len(words); i++ {
		words[i] = -1
	}
	if words[127] != -1 {
		panic("stack setup")
	}
}

func namedFalseTuple() (first, second bool) { return }

func checkNamedFalseTuple() bool {
	first, second := namedFalseTuple()
	return first || second
}

func appMain() int {
	poisonBoolTupleFrame()
	if checkNamedFalseTuple() {
		panic("named boolean results copied stack padding")
	}
	print("PASS\n")
	return 0
}
