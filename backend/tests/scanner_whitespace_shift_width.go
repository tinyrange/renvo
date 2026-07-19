package main

func renvoScannerIdentity(input int) (int, bool) {
	return input, true
}

func appMain() int {
	value, ok := renvoScannerIdentity(7)
	if !ok || value != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
