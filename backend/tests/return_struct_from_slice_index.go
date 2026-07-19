package main

type renvoReturnStructItem struct {
	a int
	b int
}

func renvoReturnStructPick(items []renvoReturnStructItem, i int) renvoReturnStructItem {
	return items[i]
}

func appMain(args []string, env []string) int {
	var items []renvoReturnStructItem
	items = append(items, renvoReturnStructItem{a: 3, b: 4})
	items = append(items, renvoReturnStructItem{a: 5, b: 6})
	item := renvoReturnStructPick(items, 1)
	if item.a == 5 && item.b == 6 {
		print("PASS\n")
	}
	return 0
}
