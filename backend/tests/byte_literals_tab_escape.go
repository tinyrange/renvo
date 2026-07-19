package main

func appMain(args []string) int {
	b := '\t'
	if b != 9 {
		print("byte_literals_03 byte\n")
		return 1
	}
	print("PASS\n")
	return 0
}
