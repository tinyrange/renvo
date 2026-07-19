package main

const intLitBase = 8
const intLitDerived = intLitBase*3 + 2

func appMain(args []string) int {
	if intLitDerived != 26 {
		print("integer_literals_15 reuse\n")
		return 1
	}
	print("PASS\n")
	return 0
}
