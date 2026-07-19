package main

func appMain(args []string) int {
	b := []byte("abc")
	b[1] = 'F'
	if b[1] != 70 {
		print("byte_literals_11 store\n")
		return 1
	}
	print("PASS\n")
	return 0
}
