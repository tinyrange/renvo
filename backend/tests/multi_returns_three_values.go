package main

func renvo1002Triple() (int, int, int) {
	return 2, 3, 5
}

func appMain(args []string) int {
	a, b, c := renvo1002Triple()
	if a*100+b*10+c != 235 {
		print("RENVO-1002 three returns failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
