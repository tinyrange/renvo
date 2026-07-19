package main

type growItem struct {
	a int
	b int
}

func appMain(args []string) int {
	items := make([]growItem, 0, 2)
	i := 0
	for i < 9 {
		items = append(items, growItem{a: i, b: i + 100})
		i++
	}
	if len(items) != 9 {
		print("RENVO-1124 struct slice append growth length failed\n")
		return 1
	}
	if items[0].a != 0 || items[0].b != 100 {
		print("RENVO-1124 struct slice append growth first item failed\n")
		return 1
	}
	if items[8].a != 8 || items[8].b != 108 {
		print("RENVO-1124 struct slice append growth last item failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
