package main

const intLitReturn = 50

func intLitReturner() int { return intLitReturn + 7 }
func appMain(args []string) int {
	if intLitReturner() != 57 {
		print("integer_literals_19 return\n")
		return 1
	}
	print("PASS\n")
	return 0
}
