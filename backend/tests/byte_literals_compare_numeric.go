package main

func appMain(args []string) int {
	b := byte('B')
	if b != 66 {
		print("byte_literals_07 numeric\n")
		return 1
	}
	print("PASS\n")
	return 0
}
