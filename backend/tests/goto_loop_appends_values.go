package main

func appMain(args []string) int {
	var xs []int
	i := 0
loop:
	if i < 3 {
		xs = append(xs, i+1)
		i = i + 1
		goto loop
	}
	if len(xs) != 3 || xs[2] != 3 {
		print("RENVO-0468 goto append failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
