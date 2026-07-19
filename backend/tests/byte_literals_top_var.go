package main

var byteLit16Global byte = 'K'

func appMain(args []string) int {
	if byteLit16Global != 75 {
		print("byte_literals_16 global\n")
		return 1
	}
	print("PASS\n")
	return 0
}
