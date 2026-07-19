package main

func appMain(args []string) int {
	b := '\r'
	if b != 13 {
		print("byte_literals_04 carriage return\n")
		return 1
	}
	if b == 'r' {
		print("byte_literals_04 letter r\n")
		return 1
	}
	print("PASS\n")
	return 0
}
