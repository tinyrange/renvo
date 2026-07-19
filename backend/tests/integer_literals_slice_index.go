package main

const intLitIndex = 2

func appMain(args []string) int {
	b := []byte("abc")
	b[intLitIndex] = 'X'
	if b[2] != 88 {
		print("integer_literals_17 index\n")
		return 1
	}
	print("PASS\n")
	return 0
}
