package main

func appMain(args []string) int {
	const (
		A    = iota
		iota = iota
		B
		C
	)
	if A != 0 || B != 1 || C != 1 {
		return 1
	}
	print("PASS\n")
	return 0
}
