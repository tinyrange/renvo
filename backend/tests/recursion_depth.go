package main

func renvo0524Depth(n int) int {
	if n == 0 {
		return 0
	}
	return 1 + renvo0524Depth(n-1)
}

func appMain(args []string) int {
	// depth variant with whitespace around the call
	value := renvo0524Depth(
		12)
	if value != 12 {
		print("RENVO-0524 depth failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
