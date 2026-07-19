package main

func appMain(args []string) int {
	b := []byte("P")
	b = append(b, 'Q')
	if len(b) != 2 || b[1] != 'Q' {
		print("byte_literals_20 append\n")
		return 1
	}
	print("PASS\n")
	return 0
}
