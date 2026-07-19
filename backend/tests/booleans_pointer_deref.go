package main

func appMain(args []string) int {
	ok := true
	p := &ok
	if !*p {
		print("booleans_20 ptr\n")
		return 1
	}
	print("PASS\n")
	return 0
}
