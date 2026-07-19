package main

type renvoAE31Number int

func appMain(args []string) int {
	dest := []renvoAE31Number{1}
	source := []renvoAE31Number{2, 4}
	dest = append(dest, source...)
	total := int(dest[0]) + int(dest[1]) + int(dest[2])
	if len(dest) != 3 || total != 7 {
		print("append_expansion_named_element_type failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
