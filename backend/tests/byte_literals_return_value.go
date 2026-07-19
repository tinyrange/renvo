package main

func byteLit10() byte { return 'E' }
func appMain(args []string) int {
	if byteLit10() != 69 {
		print("byte_literals_10 return\n")
		return 1
	}
	print("PASS\n")
	return 0
}
