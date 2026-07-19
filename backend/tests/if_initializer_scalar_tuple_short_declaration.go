package main

func scalarIfInitializerPair() (int, bool) {
	return 1, true
}

func appMain(args []string) int {
	if value, ok := scalarIfInitializerPair(); ok {
		_ = value
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
