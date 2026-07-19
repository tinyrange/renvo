package main

func renvo0439Done(x int) bool { return x >= 3 }
func appMain(args []string) int {
	i := 0
	for {
		if renvo0439Done(i) {
			break
		}
		i = i + 1
	}
	if i != 3 {
		print("RENVO-0439 helper break failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
