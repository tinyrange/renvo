package main

func renvo0482First() int { return renvo0482Second() + 1 }
func appMain(args []string) int {
	if renvo0482First() != 9 {
		print("RENVO-0482 chain later failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
func renvo0482Second() int { return 8 }
