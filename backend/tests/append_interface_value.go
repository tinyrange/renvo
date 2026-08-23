package main

type appendInterfaceValue interface {
	Value() int
}

type appendInterfaceItem struct{ value int }

func (item *appendInterfaceItem) Value() int { return item.value }

func appMain(args []string) int {
	var values []appendInterfaceValue
	var value appendInterfaceValue
	value = &appendInterfaceItem{value: 42}
	values = append(values, value)
	if len(values) != 1 || values[0].Value() != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
