package main

func appMain(args []string) int {
	b := 'A'
	if b != 65 {
		print("byte_literals_01 byte\n")
		return 1
	}
	print("PASS\n")
	return 0
}
