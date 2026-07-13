package main

func rtgLegacy53Pair() (int, int) {
	return 2, 3
}

func rtgLegacy53Add(a, b int) int {
	return a + b
}

func appMain(args []string) int {
	if rtgLegacy53Add(rtgLegacy53Pair()) != 5 {
		print("grouped_parameters_direct_tuple_call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
