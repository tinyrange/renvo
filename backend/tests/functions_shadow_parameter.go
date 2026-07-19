package main

func renvo0495Shadow(x int) int {
	if x > 0 {
		x := 9
		return x
	}
	return x
}
func appMain(args []string) int {
	if renvo0495Shadow(2) != 9 {
		print("RENVO-0495 shadow param failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
