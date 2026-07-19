package main

func renvo0508Bump(p *int, n int) {
	if n == 0 {
		return
	}
	*p = *p + 2
	renvo0508Bump(p, n-1)
}

func appMain(args []string) int {
	value := 1
	for value < 2 {
		renvo0508Bump(&value, 3)
	}
	if value != 7 {
		print("RENVO-0508 pointer recursion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
