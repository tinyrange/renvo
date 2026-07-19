package main

func appMain(args []string) int {
	b := byte(65)
	x := 0
	x = int(b)
	if x != 65 {
		print("RENVO-0346 conversion assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
