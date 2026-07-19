package main

func appMain(args []string) int {
	a := 3
	b := 9
	a++
	b--
	a++
	b--
	checksum := a*10 + b
	if checksum != 57 {
		print("RENVO-INCDEC-012 mixed checksum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
