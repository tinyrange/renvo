package main

var calls int

func source() []int {
	calls++
	return []int{1}
}

func appMain(args []string) int {
	values := append(source(), 2)
	if calls != 1 || len(values) != 2 || values[1] != 2 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
