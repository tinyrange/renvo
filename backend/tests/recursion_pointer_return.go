package main

func renvo0519Pick(n int, p *int) *int {
	if n == 0 {
		return p
	}
	*p += 1
	return renvo0519Pick(n-1, p)
}

func appMain(args []string) int {
	value := 2
	out := renvo0519Pick(4, &value)
	if *out != 6 {
		print("RENVO-0519 pointer return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
