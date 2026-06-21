package main

type rtgSL8Bag struct {
	values []int
}

func appMain(args []string) int {
	bag := rtgSL8Bag{values: []int{5, 7, 9}}
	if len(bag.values) != 3 {
		print("slice_literals_field_initializer length failed\n")
		return 1
	}
	if bag.values[0]+bag.values[2] != 14 {
		print("slice_literals_field_initializer value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
