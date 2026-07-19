package main

type aliasBox struct {
	values []int
}

func wrapAlias(values []int) aliasBox {
	var box aliasBox
	box.values = values
	return box
}

func appMain(args []string) int {
	values := make([]int, 0, 4)
	values = append(values, 1)
	box := wrapAlias(values)

	values[0] = 7
	if box.values[0] != 7 {
		print("RENVO-1121 returned struct slice lost source alias\n")
		return 1
	}

	box.values[0] = 9
	if values[0] != 9 {
		print("RENVO-1121 returned struct slice lost returned alias\n")
		return 1
	}

	print("PASS\n")
	return 0
}
