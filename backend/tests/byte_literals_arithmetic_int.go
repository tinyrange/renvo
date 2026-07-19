package main

func appMain(args []string) int {
	b := 'J'
	if int(b)+2 != 76 {
		print("byte_literals_15 arith\n")
		return 1
	}
	print("PASS\n")
	return 0
}
