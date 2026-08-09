package main

func appMain(args []string) int {
	if len(args) != 1 || len(args[0]) == 0 {
		return 1
	}
	print("PASS\n")
	return 0
}
