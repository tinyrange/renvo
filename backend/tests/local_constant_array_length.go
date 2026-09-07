package main

const size = 1

func appMain(args []string) int {
	const size = size + 1
	a := [size]int{1, 7}
	if len(a) != 2 || a[1] != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
