package main

type mapSliceBox struct {
	values map[string]int
}

func appMain() int {
	first := map[string]int{"a": 1}
	rows := []map[string]int{first, map[string]int{"b": 2}}
	box := mapSliceBox{values: rows[1]}
	if len(first) == 1 && len(rows[0]) == 1 && len(rows[1]) == 1 && len(box.values) == 1 && rows[0]["a"] == 1 && box.values["b"] == 2 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
