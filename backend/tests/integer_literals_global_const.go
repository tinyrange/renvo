package main

const intLitGlobal = 91

func appMain(args []string) int {
	if intLitGlobal != 91 {
		print("integer_literals_25 global\n")
		return 1
	}
	print("PASS\n")
	return 0
}
