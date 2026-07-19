package main

func renvo0483A() int { return renvo0483B() + 1 }
func renvo0483B() int { return renvo0483C() + 2 }
func renvo0483C() int { return 3 }
func appMain(args []string) int {
	if renvo0483A() != 6 {
		print("RENVO-0483 chain failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
