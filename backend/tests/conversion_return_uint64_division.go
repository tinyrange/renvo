package main

func pagesForConversionReturn(bytes uint64) uintptr {
	return uintptr((bytes + 4095) / 4096)
}

func appMain(args []string) int {
	if pagesForConversionReturn(0) != 0 || pagesForConversionReturn(4096) != 1 || pagesForConversionReturn(8193) != 3 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
