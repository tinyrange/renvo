package main

func appMain(args []string) int {
	var left uint32 = 1
	var unsignedRight uint32 = 0x80000000
	var signedRight int32 = -2147483648
	leftCount := uint32(30)
	rightCount := uint32(31)
	if left<<leftCount != 1073741824 {
		print("FAIL left\n")
		return 1
	}
	if unsignedRight>>rightCount != 1 {
		print("FAIL unsigned right\n")
		return 1
	}
	if signedRight>>rightCount != -1 {
		print("FAIL signed right\n")
		return 1
	}
	print("PASS\n")
	return 0
}
