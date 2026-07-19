package main

func renvo0533Wide(n int) int64 {
	for n < 5 {
		n = n + 1
	}
	return int64(n * 3)
}

func appMain(args []string) int {
	if renvo0533Wide(2) != 15 {
		print("RENVO-0533 int64 return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
