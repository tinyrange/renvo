package main

func appMain(args []string) int {
	b := '\''
	if b != 39 {
		print("byte_literals_04 byte\n")
		return 1
	}
	print("PASS\n")
	return 0
}
