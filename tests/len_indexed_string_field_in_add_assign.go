package main

type rtgLenIndexedStringList struct {
	values []string
}

func appMain() int {
	list := rtgLenIndexedStringList{}
	list.values = append(list.values, "abc")
	list.values = append(list.values, "de")
	total := 1
	for i := 0; i < len(list.values); i++ {
		total += len(list.values[i]) + 2
	}
	if total != 10 {
		print("len_indexed_string_field_in_add_assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
