package main

func appMain(args []string) int {
	if len(args) == 0 {
		print("missing args\n")
		return 1
	}
	print("PASS\n")
	return 0
}
