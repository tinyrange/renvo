package main

func rtgScannerIdentity(input int) (int, bool) {
	return input, true
}

func appMain() int {
	value, ok := rtgScannerIdentity(7)
	if !ok || value != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
