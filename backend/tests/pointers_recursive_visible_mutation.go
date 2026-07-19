package main

func renvo0650After(p *int, n int) {
	if n == 0 {
		*p = *p + 10
		return
	}
	renvo0650After(p, n-1)
	*p = *p + 1
}

func appMain(args []string) int {
	value := 0
	renvo0650After(&value, 3)
	if value != 13 {
		print("RENVO-0650 recursive visible mutation failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
