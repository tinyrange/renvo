package main

func renvoWasmHelperBlockSame(a string, b string) bool {
	return a == b
}

func appMain(args []string) int {
	ok := renvoWasmHelperBlockSame("renvo", "renvo")
	if ok {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
