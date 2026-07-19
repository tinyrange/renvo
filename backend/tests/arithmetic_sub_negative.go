package main

func appMain(args []string) int {
	a := 8
	b := 50
	if a-b != -42 {
		print("arithmetic_04 neg\n")
		return 1
	}
	print("PASS\n")
	return 0
}
