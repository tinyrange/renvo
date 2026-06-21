package main

func appMain(args []string) int {
	dest := []int{1}
	first := []int{2, 3}
	second := []int{4, 5}
	dest = append(dest, first...)
	dest = append(dest, second...)
	sum := 0
	i := 0
	for i < len(dest) {
		sum += dest[i]
		i += 1
	}
	if len(dest) != 5 || sum != 15 {
		print("append_expansion_chained_two_sources failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
