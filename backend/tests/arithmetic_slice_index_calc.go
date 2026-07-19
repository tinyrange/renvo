package main

func appMain(args []string) int {
	b := []byte("abcd")
	i := 1
	if b[i*2+1] != 'd' {
		print("arithmetic_16 index\n")
		return 1
	}
	print("PASS\n")
	return 0
}
