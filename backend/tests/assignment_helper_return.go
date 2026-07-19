package main

func renvo0345Value() int { return 12 }

func appMain(args []string) int {
	x := 0
	x = renvo0345Value()
	if x != 12 {
		print("RENVO-0345 helper assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
