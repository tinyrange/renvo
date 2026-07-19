package main

func appMain(args []string) int {
	outer := 0
	innerHits := 0
	for {
		for {
			innerHits = innerHits + 1
			break
		}
		outer = outer + 1
		if outer == 3 {
			break
		}
	}
	if outer != 3 || innerHits != 3 {
		print("RENVO-0432 inner break failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
