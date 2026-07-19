package main

func appMain(args []string) int {
	b := '\\'
	if b != 92 {
		print("byte_literals_05 byte\n")
		return 1
	}
	print("PASS\n")
	return 0
}
