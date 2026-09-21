package main

func appMain(args []string) int {
	const true = 3
	const false = 2
	var values [true]int
	var more [1 + false]int
	if len(values) != 3 || len(more) != 3 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
