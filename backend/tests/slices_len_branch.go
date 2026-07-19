package main

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 1)
	xs = append(xs, 2)
	if len(xs) == 2 {
		print("PASS\n")
		return 0
	}
	print("RENVO-0573 len branch failed\n")
	return 1
}
