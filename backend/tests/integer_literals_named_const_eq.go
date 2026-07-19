package main

const intLitA = 77
const intLitB = intLitA

func appMain(args []string) int {
	if intLitA != intLitB {
		print("integer_literals_23 consteq\n")
		return 1
	}
	print("PASS\n")
	return 0
}
