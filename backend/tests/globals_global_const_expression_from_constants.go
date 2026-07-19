package main

const renvo0682Base = 11
const renvo0682Value = renvo0682Base*3 + 1

func appMain(args []string) int {
	if renvo0682Value != 34 {
		print("RENVO-0682 const expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
