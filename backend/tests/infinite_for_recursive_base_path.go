package main

func renvo0450Walk(n int) int {
	if n == 0 {
		for {
			break
		}
		return 1
	}
	return renvo0450Walk(n-1) + 1
}
func appMain(args []string) int {
	if renvo0450Walk(3) != 4 {
		print("RENVO-0450 recursive infinite base failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
