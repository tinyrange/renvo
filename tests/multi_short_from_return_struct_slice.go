package main

type rtg1040Bag struct {
	value int
}

func rtg1040Make() (rtg1040Bag, []int) {
	var xs []int
	xs = append(xs, 6)
	return rtg1040Bag{value: 5}, xs
}

func appMain(args []string) int {
	bag, xs := rtg1040Make()
	if bag.value+xs[0] != 11 {
		print("RTG-1040 short struct slice return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
