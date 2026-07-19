package main

func renvo0489Read(p *int) int { return *p }
func appMain(args []string) int {
	x := 14
	if renvo0489Read(&x) != 14 {
		print("RENVO-0489 pointer param failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
