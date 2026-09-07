package main

func appMain(args []string) int {
	const reset = iota
	const next = iota + 1
	if reset != 0 || next != 1 {
		return 1
	}
	print("PASS\n")
	return 0
}
