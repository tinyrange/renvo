package main

func appMain(args []string) int {
	x := int('T')
	x += 2
	b := byte(x)
	if b != 'V' {
		print("byte_literals_23 compound\n")
		return 1
	}
	print("PASS\n")
	return 0
}
