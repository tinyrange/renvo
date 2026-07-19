package main

func appMain(args []string) int {
	var xs []int
	for i := 0; i < 3; i = i + 1 {
		xs = append(xs, i+10)
	}
	if xs[0] != 10 {
		print("RENVO-0559 first index failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
