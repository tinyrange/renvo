package main

type rtg1048Record struct {
	base int
}

func rtg1048Build() (rtg1048Record, []int) {
	var xs []int
	xs = append(xs, 3)
	xs = append(xs, 4)
	return rtg1048Record{base: 5}, xs
}

func rtg1048Use(r rtg1048Record, xs []int) int {
	return r.base + xs[0] + xs[1]
}

func appMain(args []string) int {
	if rtg1048Use(rtg1048Build()) != 12 {
		print("RTG-1048 direct struct slice args failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
