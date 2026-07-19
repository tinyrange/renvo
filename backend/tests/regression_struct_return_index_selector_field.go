package main

type token struct {
	text string
}

type info struct {
	name string
	pos  int
}

func pick(tokens []token, index int) (info, bool) {
	return info{
		name: tokens[index].text,
		pos:  index,
	}, true
}

func appMain() int {
	tokens := []token{{text: "PASS\n"}}
	got, ok := pick(tokens, 0)
	if ok && got.name == "PASS\n" && got.pos == 0 {
		print(got.name)
		return 0
	}
	print("FAIL\n")
	return 1
}
