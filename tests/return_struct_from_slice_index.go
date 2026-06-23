package main

type rtgReturnStructItem struct {
	a int
	b int
}

func rtgReturnStructPick(items []rtgReturnStructItem, i int) rtgReturnStructItem {
	return items[i]
}

func appMain(args []string, env []string) int {
	var items []rtgReturnStructItem
	items = append(items, rtgReturnStructItem{a: 3, b: 4})
	items = append(items, rtgReturnStructItem{a: 5, b: 6})
	item := rtgReturnStructPick(items, 1)
	if item.a == 5 && item.b == 6 {
		print("PASS\n")
	}
	return 0
}
