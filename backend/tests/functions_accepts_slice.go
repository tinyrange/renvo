package main

func renvo0490Len(xs []int) int { return len(xs) }
func appMain(args []string) int {
	var xs []int
	xs = append(xs, 1)
	if renvo0490Len(xs) != 1 {
		print("RENVO-0490 slice param failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
