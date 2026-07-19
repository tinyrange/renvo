package main

var renvo0476Seen int

func renvo0476Mark() { renvo0476Seen = 1 }
func appMain(args []string) int {
	renvo0476Mark()
	if renvo0476Seen != 1 {
		print("RENVO-0476 void helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
