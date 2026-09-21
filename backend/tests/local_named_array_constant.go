package main

func appMain(args []string) int {
	const count = 2
	type A [count]int
	var a A
	a[1] = 7
	if len(a) != 2 || a[1] != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
