package main

func renvo0542Jump(xs []int) int {
	xs = append(xs, 7)
	goto done
done:
	return len(xs)
}

func appMain(args []string) int {
	var xs []int
	if renvo0542Jump(xs) != 1 {
		print("RENVO-0542 goto return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
