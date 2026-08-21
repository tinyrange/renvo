package main

type interfaceSliceTypeSwitchBox struct {
	value any
}

type interfaceSliceTypeSwitchEntry struct {
	key   string
	value any
}

type interfaceSliceTypeSwitchMap struct {
	entries []interfaceSliceTypeSwitchEntry
}

func interfaceSliceTypeSwitchValue(box *interfaceSliceTypeSwitchBox) any {
	return box.value
}

func interfaceSliceTypeSwitchMake(first string, firstValue any, second string, secondValue any) *interfaceSliceTypeSwitchBox {
	_ = first
	_ = second
	_ = secondValue
	box := &interfaceSliceTypeSwitchBox{}
	target := &box.value
	*target = firstValue
	return box
}

func interfaceSliceTypeSwitchMatches(value any) bool {
	switch items := value.(type) {
	case []any:
		return len(items) == 3
	}
	return false
}

func interfaceSliceTypeSwitchRecursive(value any, next any) bool {
	switch item := value.(type) {
	case string:
		_ = item
		return interfaceSliceTypeSwitchRecursive(next, nil)
	case []any:
		return len(item) == 3
	}
	return false
}

func interfaceSliceTypeSwitchMakeDynamic() any {
	return []any{1, true, nil}
}

func interfaceSliceTypeSwitchGet(mapping []*interfaceSliceTypeSwitchMap, index int) any {
	return mapping[0].entries[index].value
}

func interfaceSliceTypeSwitchRef(mapping []*interfaceSliceTypeSwitchMap, index int) *any {
	return &mapping[0].entries[index].value
}

func appMain() int {
	box := interfaceSliceTypeSwitchMake("z", []any{1, true, nil}, "a", "x")
	if !interfaceSliceTypeSwitchMatches(interfaceSliceTypeSwitchValue(box)) {
		print("FAIL\n")
		return 1
	}
	entries := make([]interfaceSliceTypeSwitchEntry, 2)
	entries[0].key = "a"
	entries[0].value = "x"
	entries[1].key = "z"
	storage := &interfaceSliceTypeSwitchMap{entries: entries}
	mapping := []*interfaceSliceTypeSwitchMap{storage}
	*interfaceSliceTypeSwitchRef(mapping, 1) = []any{1, true, nil}
	if !interfaceSliceTypeSwitchMatches(interfaceSliceTypeSwitchGet(mapping, 1)) {
		print("FAIL\n")
		return 1
	}
	if !interfaceSliceTypeSwitchRecursive("x", interfaceSliceTypeSwitchGet(mapping, 1)) {
		print("FAIL\n")
		return 1
	}
	if !interfaceSliceTypeSwitchMatches(interfaceSliceTypeSwitchMakeDynamic()) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
