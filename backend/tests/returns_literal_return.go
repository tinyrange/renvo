package main

var renvo0528Want int = 42

func renvo0528Value() int {
	return 42
}

func appMain(args []string) int {
	if renvo0528Value() != renvo0528Want {
		print("RENVO-0528 literal return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
