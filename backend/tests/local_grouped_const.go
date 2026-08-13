package main

func appMain(args []string) int {
	const (
		blah  = 20
		blah2 = 22
	)
	if blah+blah2 != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
