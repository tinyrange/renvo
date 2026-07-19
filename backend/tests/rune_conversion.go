package main

func convertRune(value byte) rune {
	return rune(value)
}

func appMain() int {
	if convertRune(233) == 'é' && '\u4e16' == 19990 && '\U0001F600' == 128512 && '\x41' == 'A' && '\101' == 'A' {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
