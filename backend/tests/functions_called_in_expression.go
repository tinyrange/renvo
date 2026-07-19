package main

func renvo0498A() int { return 5 }
func appMain(args []string) int {
	x := renvo0498A()*2 + 3
	if x != 13 {
		print("RENVO-0498 expression call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
