package main

type renvoCopyOverlapItem struct {
	kind   string
	name   string
	text   string
	x      int
	y      int
	width  int
	height int
}

func appMain(args []string) int {
	items := []renvoCopyOverlapItem{
		{kind: "label", name: "first", text: "First"},
		{kind: "button", name: "second", text: "Second"},
		{kind: "textBox", name: "third", text: "Third"},
		{kind: "panel", name: "fourth", text: "Fourth"},
		{},
	}
	count := copy(items[1:], items[:4])
	if count != 4 || items[1].name != "first" || items[2].name != "second" || items[3].name != "third" || items[4].name != "fourth" {
		return 1
	}
	print("PASS\n")
	return 0
}
