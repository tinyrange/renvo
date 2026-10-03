package main

func fitsSignedWord(v int) bool {
	return v >= -2147483647 && v <= 2147483647
}

func compareSignedWords(a int, b int) bool {
	if !(a > b) || a < b || !(a >= b) || a <= b || a == b || !(a != b) {
		return false
	}
	greater := a > b
	less := a < b
	return greater && !less
}

func appMain() int {
	if !fitsSignedWord(134217728) || !fitsSignedWord(-134217728) {
		panic("signed word bounds")
	}
	if !compareSignedWords(2147483647, -2147483647) || !compareSignedWords(0, -2147483647) || !compareSignedWords(2147483647, 0) {
		panic("signed comparison")
	}
	var minimum int = -2147483647 - 1
	if minimum >= 1 || minimum > -1 || !(minimum < 1) || !(minimum <= -1) {
		panic("signed immediate comparison")
	}
	print("PASS\n")
	return 0
}
