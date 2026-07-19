package main

func renvo0663Convert(n int) int64 {
	if n == 0 {
		return int64(0)
	}
	return int64(1) + renvo0663Convert(n-1)
}

func appMain(args []string) int {
	if renvo0663Convert(5) != 5 {
		print("RENVO-0663 int int64 conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
