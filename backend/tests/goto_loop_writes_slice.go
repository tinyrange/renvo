package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 0)
	xs = append(xs, 0)
	i := 0
loop:
	if i < len(xs) {
		xs[i] = i + 11
		i = i + 1
		goto loop
	}
	if xs[1] != 12 {
		print("RENVO-0470 goto slice write failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
