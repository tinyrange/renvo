package main

func renvo0510Caller(n int) int {
	for {
		if n <= 0 {
			break
		}
		return renvo0510Later(n)
	}
	return 0
}

func appMain(args []string) int {
	if renvo0510Caller(4) != 10 {
		print("RENVO-0510 late helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func renvo0510Later(n int) int {
	if n == 0 {
		return 0
	}
	return n + renvo0510Later(n-1)
}
