package main

// renvo:linkstatic /usr/lib/librenvo_missing.dylib,renvo_missing
func targetForeignCall() int { return 7 }

func appMain() int {
	if targetForeignCall() != 7 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
