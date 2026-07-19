package main

func renvo0565Make(p *int) []int {
	var xs []int
	xs = append(xs, *p)
	return xs
}

func appMain(args []string) int {
	value := 13
	xs := renvo0565Make(&value)
	if len(xs) != 1 || xs[0] != 13 {
		print("RENVO-0565 slice return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
