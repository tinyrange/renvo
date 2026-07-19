package main

type renvoAE30Entry struct {
	key   int
	value int
}

func appMain(args []string) int {
	dest := []renvoAE30Entry{{key: 1, value: 2}}
	source := []renvoAE30Entry{{key: 3, value: 4}, {key: 5, value: 6}}
	dest = append(dest, source...)
	if len(dest) != 3 {
		print("append_expansion_struct_slice length failed\n")
		return 1
	}
	if dest[0].key+dest[1].value+dest[2].key != 10 {
		print("append_expansion_struct_slice value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
