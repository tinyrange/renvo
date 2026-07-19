package main

func appMain(args []string) int {
	var xs []int
	for i := 0; i < 4; i = i + 1 {
		if i == 2 {
			continue
		}
		xs = append(xs, i)
	}
	xs[1] = 22
	if xs[1] != 22 || len(xs) != 3 {
		print("RENVO-0561 int index assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
