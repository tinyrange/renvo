package main

func renvo0500A() int      { return 7 }
func renvo0500B(x int) int { return x + 18 }
func appMain(args []string) int {
	if renvo0500B(renvo0500A()) != 25 {
		print("RENVO-0500 direct result failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
