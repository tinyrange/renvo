package main

type rtgSL5Pair struct {
	left  int
	right byte
}

func appMain(args []string) int {
	items := []rtgSL5Pair{{left: 3, right: 'a'}, {left: 4, right: 'b'}}
	if len(items) != 2 {
		print("slice_literals_struct_elements length failed\n")
		return 1
	}
	if items[0].left+items[1].left+int(items[1].right) != 7+int('b') {
		print("slice_literals_struct_elements checksum failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
