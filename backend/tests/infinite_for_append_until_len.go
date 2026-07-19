package main

func appMain(args []string) int {
	var xs []int
	for {
		if len(xs) == 3 {
			break
		}
		xs = append(xs, len(xs)+1)
	}
	if xs[2] != 3 {
		print("RENVO-0436 append infinite failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
