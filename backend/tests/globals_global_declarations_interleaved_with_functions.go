package main

var renvo0697Value int = 5

func renvo0697Get() int {
	return renvo0697Value
}

const renvo0697Add = 6

func appMain(args []string) int {
	if renvo0697Get()+renvo0697Add != 11 {
		print("RENVO-0697 interleaved globals failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
