package main

func appMain(args []string) int {
	var xs []int
	for {
		xs = append(xs, len(xs)*2)
		if len(xs) == 4 {
			break
		}
	}
	if xs[3] != 6 {
		print("RENVO-0560 last index failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
