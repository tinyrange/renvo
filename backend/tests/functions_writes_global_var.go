package main

var renvo0494Global int

func renvo0494Set() { renvo0494Global = 20 }
func appMain(args []string) int {
	renvo0494Set()
	if renvo0494Global != 20 {
		print("RENVO-0494 global var failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
