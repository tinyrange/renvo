package main

func appMain(args []string) int {
	var xs []int
	for i := 0; i < 3; i = i + 1 {
		xs = append(xs, i+2)
	}
	if len(xs) != 3 || xs[2] != 4 {
		print("RENVO-0412 append body failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
