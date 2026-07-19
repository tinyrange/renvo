package main

const intLitArg = 11

func intLitTriple(x int) int { return x * 3 }
func appMain(args []string) int {
	if intLitTriple(intLitArg) != 33 {
		print("integer_literals_20 func\n")
		return 1
	}
	print("PASS\n")
	return 0
}
