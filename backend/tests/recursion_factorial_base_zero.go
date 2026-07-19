package main

func renvo0501Fact(n int) int {
	if n == 0 {
		return 1
	}
	return n * renvo0501Fact(n-1)
}

func appMain(args []string) int {
	if renvo0501Fact(0) != 1 {
		print("RENVO-0501 factorial base failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
