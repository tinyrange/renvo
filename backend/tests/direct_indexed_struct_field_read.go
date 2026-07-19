package main

type item struct {
	start int
	end   int
}

func valueAt(items []item, i int) int {
	return items[i].start + items[i].end
}

func appMain() int {
	items := []item{{start: 12, end: 30}}
	if valueAt(items, 0) != 42 {
		print("direct_indexed_struct_field_read failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
