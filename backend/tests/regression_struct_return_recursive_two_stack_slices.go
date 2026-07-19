package main

type token struct {
	Text string
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

func lookup(table []entry, name string) info {
	for i := 0; i < len(table); i++ {
		item := table[i]
		if item.name == name {
			return item.info
		}
	}
	return info{}
}

func infer(tokens []token, start int, end int, types []entry, functionResults []entry) info {
	if tokens[start].Text == "(" {
		return infer(tokens, start+1, end-1, types, functionResults)
	}
	if tokens[start].Text == "call" {
		return lookup(functionResults, tokens[start].Text)
	}
	return lookup(types, tokens[start].Text)
}

func appMain() int {
	tokens := []token{{Text: "("}, {Text: "call"}, {Text: ")"}}
	types := []entry{{name: "value", info: info{qualifier: "FAIL", name: "\n", pointer: false}}}
	functionResults := []entry{{name: "call", info: info{qualifier: "PASS", name: "\n", pointer: true}}}
	got := infer(tokens, 0, len(tokens), types, functionResults)
	if got.pointer && got.qualifier == "PASS" && got.name == "\n" {
		print(got.qualifier)
		print(got.name)
		return 0
	}
	print("FAIL\n")
	return 1
}
