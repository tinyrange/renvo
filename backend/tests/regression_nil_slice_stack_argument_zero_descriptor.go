package main

type token struct {
	Kind   int
	Text   string
	Start  int32
	End    int32
	Line   int32
	Column int32
}

type info struct {
	qualifier string
	name      string
	pointer   bool
}

type entry struct {
	name string
	info info
}

type table []entry

func lookup(items table, name string) info {
	for i := 0; i < len(items); i++ {
		item := items[i]
		if item.name == name {
			return item.info
		}
	}
	return info{}
}

func infer(tokens []token, start int, end int, types table, functionResults table) info {
	if tokens[start].Text == "call" {
		return lookup(functionResults, tokens[start].Text)
	}
	return lookup(types, tokens[start].Text)
}

func dirty(a int, b int, c int, d int, e int, f int, g int, h int, i int) int {
	return g + h + i
}

func inferWithoutFunctions(tokens []token, start int, end int, types table) info {
	if dirty(1, 2, 3, 4, 5, 6, 700, 800, 900) == 0 {
		return info{qualifier: "FAIL", name: "\n", pointer: true}
	}
	return infer(tokens, start, end, types, nil)
}

func appMain() int {
	tokens := []token{{Text: "call"}}
	types := table{{name: "value", info: info{qualifier: "FAIL", name: "\n", pointer: true}}}
	got := inferWithoutFunctions(tokens, 0, len(tokens), types)
	if got.qualifier == "" && got.name == "" && !got.pointer {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
