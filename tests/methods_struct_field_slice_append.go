package main

type rtgMD40Bag struct {
	values []int
}

func (b *rtgMD40Bag) Add(n int) {
	b.values = append(b.values, n)
}

func appMain(args []string) int {
	bag := rtgMD40Bag{values: []int{1}}
	bag.Add(2)
	bag.Add(4)
	if len(bag.values) != 3 {
		print("methods_struct_field_slice_append length failed\n")
		return 1
	}
	if bag.values[0]+bag.values[1]+bag.values[2] != 7 {
		print("methods_struct_field_slice_append value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
