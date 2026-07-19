package main

type sliceFieldByValueItem struct {
	value int
}

type sliceFieldByValueParser struct {
	items []sliceFieldByValueItem
	pos   int
}

func sliceFieldByValueRead(p sliceFieldByValueParser) int {
	item := p.items[0]
	return item.value + len(p.items) + p.pos
}

func appMain(args []string) int {
	var items []sliceFieldByValueItem
	items = append(items, sliceFieldByValueItem{value: 5})
	p := sliceFieldByValueParser{items: items, pos: 3}
	if sliceFieldByValueRead(p) != 9 {
		print("slice field by value argument failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
