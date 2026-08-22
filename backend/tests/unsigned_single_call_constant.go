package main

func multipliedSizeOverflows(factor uint64, size uint64) bool {
	return factor > uint64(18446744073709551615)/size
}

func appMain(args []string) int {
	if multipliedSizeOverflows(1, 16) {
		return 1
	}
	print("PASS\n")
	return 0
}
