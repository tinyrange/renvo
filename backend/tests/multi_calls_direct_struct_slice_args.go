package main

type renvo1048Record struct {
	base int
}

func renvo1048Build() (renvo1048Record, []int) {
	var xs []int
	xs = append(xs, 3)
	xs = append(xs, 4)
	return renvo1048Record{base: 5}, xs
}

func renvo1048Use(r renvo1048Record, xs []int) int {
	return r.base + xs[0] + xs[1]
}

func appMain(args []string) int {
	if renvo1048Use(renvo1048Build()) != 12 {
		print("RENVO-1048 direct struct slice args failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
