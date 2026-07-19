package main

type bucket struct {
	values []int
}

func appMain(args []string) int {
	var b bucket
	b.values = append(b.values, 1)
	b.values[0] = 7
	if b.values[0] == 7 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
