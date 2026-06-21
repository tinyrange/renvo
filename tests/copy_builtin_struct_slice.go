package main

type rtgCP24Rec struct {
	id  int
	tag byte
}

func appMain(args []string) int {
	source := []rtgCP24Rec{{id: 2, tag: 'x'}, {id: 5, tag: 'y'}}
	dest := make([]rtgCP24Rec, 2)
	n := copy(dest, source)
	if n != 2 {
		print("copy_builtin_struct_slice count failed\n")
		return 1
	}
	if dest[0].id+dest[1].id != 7 || dest[1].tag != 'y' {
		print("copy_builtin_struct_slice value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
