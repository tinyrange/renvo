package main

type rtgMK16Cell struct {
	value int
	flag  bool
}

func appMain(args []string) int {
	cells := make([]rtgMK16Cell, 2)
	cells[0] = rtgMK16Cell{value: 2, flag: true}
	cells[1] = rtgMK16Cell{value: 5, flag: false}
	if len(cells) != 2 {
		print("make_slices_struct_elements length failed\n")
		return 1
	}
	if !cells[0].flag || cells[0].value+cells[1].value != 7 {
		print("make_slices_struct_elements value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
