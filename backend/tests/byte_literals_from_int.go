package main

func appMain(args []string) int {
	b := byte(73)
	if b != 'I' {
		print("byte_literals_14 byte\n")
		return 1
	}
	print("PASS\n")
	return 0
}
