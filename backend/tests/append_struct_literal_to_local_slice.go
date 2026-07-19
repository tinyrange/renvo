package main

type appendStructLiteralItem struct {
	a int
	b int
	c int
}

func appMain() int {
	var items []appendStructLiteralItem
	items = append(items, appendStructLiteralItem{a: 3, b: 4, c: 5})
	if len(items) != 1 {
		print("FAIL\n")
		return 1
	}
	if items[0].a != 3 {
		print("FAIL\n")
		return 1
	}
	if items[0].b != 4 {
		print("FAIL\n")
		return 1
	}
	if items[0].c != 5 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
