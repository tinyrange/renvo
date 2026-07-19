package main

func appMain(args []string) int {
	ok := 7 <= 9
	if !ok {
		print("booleans_18 cmp\n")
		return 1
	}
	print("PASS\n")
	return 0
}
