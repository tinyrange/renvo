package main

type indexedCompositeRecord struct {
	pair [2]int64
}

func appMain(args []string) int {
	result := [1]indexedCompositeRecord{{pair: [2]int64{13, 29}}}[0].pair[1]
	if result != 29 {
		print("FAIL: indexed array composite struct selector\n")
		return 1
	}
	print("PASS\n")
	return 0
}
