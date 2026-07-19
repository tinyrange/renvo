package main

func renvo0638Grow(p *int, n int) {
	if n == 0 {
		return
	}
	*p = *p + 1
	renvo0638Grow(p, n-1)
}

func appMain(args []string) int {
	value := 1
	renvo0638Grow(&value, 4)
	if value != 5 {
		print("RENVO-0638 recursive pointer mutate failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
