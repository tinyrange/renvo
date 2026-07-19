package main

func renvo0668Return(n int64) int {
	return int(n)
}

func appMain(args []string) int {
	if renvo0668Return(int64(18)) != 18 {
		print("RENVO-0668 return conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
