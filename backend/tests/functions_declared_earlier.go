package main

func renvo0481Earlier() int { return 6 }
func appMain(args []string) int {
	if renvo0481Earlier() != 6 {
		print("RENVO-0481 earlier helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
