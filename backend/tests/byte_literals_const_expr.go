package main

const byteLit17Const = 'L'

func appMain(args []string) int {
	if byteLit17Const+1 != 'M' {
		print("byte_literals_17 const\n")
		return 1
	}
	print("PASS\n")
	return 0
}
