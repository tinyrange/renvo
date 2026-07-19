package main

type renvo1040Bag struct {
	value int
}

func renvo1040Make() (renvo1040Bag, []int) {
	var xs []int
	xs = append(xs, 6)
	return renvo1040Bag{value: 5}, xs
}

func appMain(args []string) int {
	bag, xs := renvo1040Make()
	if bag.value+xs[0] != 11 {
		print("RENVO-1040 short struct slice return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
