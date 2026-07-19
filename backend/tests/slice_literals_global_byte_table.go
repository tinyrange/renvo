package main

var renvoSL7Table = []byte{'P', 'A', 'S', 'S'}

func renvoSL7At(i int) byte {
	return renvoSL7Table[i]
}

func appMain(args []string) int {
	if len(renvoSL7Table) != 4 {
		print("slice_literals_global_byte_table length failed\n")
		return 1
	}
	if renvoSL7At(0) != 'P' || renvoSL7At(3) != 'S' {
		print("slice_literals_global_byte_table value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
