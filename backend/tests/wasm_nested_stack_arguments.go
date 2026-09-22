package main

type callWords struct{ a, b, c, d, e, f, g, h int }

func innerWords(v callWords) int { return v.a + v.h }
func outerWords(v callWords, n int) bool {
	return v.a == 1 && v.b == 2 && v.c == 3 && v.d == 4 && v.e == 5 && v.f == 6 && v.g == 7 && v.h == 8 && n == 30
}
func middleWords(v callWords, n int) int { return v.a + v.h + n }
func appMain(args []string) int {
	outer := callWords{1, 2, 3, 4, 5, 6, 7, 8}
	inner := callWords{10, 11, 12, 13, 14, 15, 16, 20}
	if !outerWords(outer, innerWords(inner)) {
		return 1
	}
	if !outerWords(outer, middleWords(callWords{}, innerWords(inner))) {
		return 2
	}
	print("PASS\n")
	return 0
}
