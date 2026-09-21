package main

func appMain(args []string) int {
	count := 0
	for _, row := range [][]string{{"one"}, {}, {"two", "three"}} {
		count += len(row)
	}
	values := [][][]int{{{1, 2}, {}}, {{3}}}
	empty := values[0][1]
	if count != 3 || values[0][0][1] != 2 || values[1][0][0] != 3 || empty == nil {
		return 1
	}
	print("PASS\n")
	return 0
}
