package main

func renvo0543Base(n int) int {
	if n == 0 {
		return int(byte(5))
	}
	return renvo0543Base(n-1) + 1
}

func appMain(args []string) int {
	if renvo0543Base(0) != 5 {
		print("RENVO-0543 recursive base return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
