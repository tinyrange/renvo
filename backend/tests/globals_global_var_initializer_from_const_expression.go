package main

const renvo0699A = 8
const renvo0699B = 9

var renvo0699Value int = renvo0699A*renvo0699B + 1

func appMain(args []string) int {
	if renvo0699Value != 73 {
		print("RENVO-0699 global const init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
