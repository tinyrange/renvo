package main

func appMain(args []string) int {
	var xs []int
	for i := 0; i < 3; i = i + 1 {
		for j := 0; j < 2; j = j + 1 {
			xs = append(xs, i+j)
		}
	}
	if len(xs) == 6 && xs[5] == 3 {
		print("PASS\n")
		return 0
	}
	print("RENVO-0570 nested loop fill failed\n")
	return 1
}
