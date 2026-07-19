package main

func renvoLegacy53Pair() (int, int) {
	return 2, 3
}

func renvoLegacy53Add(a, b int) int {
	return a + b
}

func appMain(args []string) int {
	if renvoLegacy53Add(renvoLegacy53Pair()) != 5 {
		print("grouped_parameters_direct_tuple_call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
