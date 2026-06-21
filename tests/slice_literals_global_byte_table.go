package main

var rtgSL7Table = []byte{'P', 'A', 'S', 'S'}

func rtgSL7At(i int) byte {
	return rtgSL7Table[i]
}

func appMain(args []string) int {
	if len(rtgSL7Table) != 4 {
		print("slice_literals_global_byte_table length failed\n")
		return 1
	}
	if rtgSL7At(0) != 'P' || rtgSL7At(3) != 'S' {
		print("slice_literals_global_byte_table value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
