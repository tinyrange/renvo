package main

func renvo0819One() int { return 1 }
func appMain(args []string) int {
	if renvo0819One() != 1 {
		print("RENVO-0819 dense helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
