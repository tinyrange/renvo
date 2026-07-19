package main

var renvo0526Hit bool

func renvo0526Mark() {
	renvo0526Hit = true
	return
}

func appMain(args []string) int {
	renvo0526Mark()
	if !renvo0526Hit {
		print("RENVO-0526 bare return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
