package main

func renvo0537Pick(flag bool) int {
	goto decide
decide:
	if flag {
		return 8
	}
	return 4
}

func appMain(args []string) int {
	if renvo0537Pick(true) != 8 {
		print("RENVO-0537 then return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
