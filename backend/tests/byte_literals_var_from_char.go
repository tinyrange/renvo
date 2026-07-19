package main

func appMain(args []string) int {
	var b byte = 'Q'
	if b != 81 {
		print("byte_literals_06 var\n")
		return 1
	}
	print("PASS\n")
	return 0
}
