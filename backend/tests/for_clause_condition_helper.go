package main

func renvo0410Limit() int { return 3 }
func appMain(args []string) int {
	sum := 0
	for i := 0; i < renvo0410Limit(); i = i + 1 {
		sum = sum + i
	}
	if sum != 3 {
		print("RENVO-0410 helper condition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
