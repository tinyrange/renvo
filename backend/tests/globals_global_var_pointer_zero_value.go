package main

var renvo0692Ptr *int

func appMain(args []string) int {
	if renvo0692Ptr != nil {
		print("RENVO-0692 pointer zero global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
