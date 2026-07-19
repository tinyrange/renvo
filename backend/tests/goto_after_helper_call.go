package main

func renvo0464Value() int { return 7 }
func appMain(args []string) int {
	x := renvo0464Value()
	goto check
check:
	if x != 7 {
		print("RENVO-0464 helper goto failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
