package main

func renvo0545A(n int) int {
	return renvo0545B(n) + 2
}

func renvo0545B(n int) int {
	return n * 3
}

func appMain(args []string) int {
	if renvo0545A(5) == 17 && renvo0545B(2) == 6 {
		print("PASS\n")
		return 0
	}
	print("RENVO-0545 nested return failed\n")
	return 1
}
