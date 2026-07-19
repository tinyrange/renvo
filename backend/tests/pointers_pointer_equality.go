package main

func appMain(args []string) int {
	value := 26
	p := &value
	q := p
	var seen []int
	if p == q {
		seen = append(seen, *q)
	}
	if len(seen) != 1 || seen[0] != 26 {
		print("RENVO-0642 pointer equality failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
