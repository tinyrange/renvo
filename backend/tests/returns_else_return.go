package main

func renvo0538Pick(flag bool) int {
	if flag {
		return 3
	} else {
		return 6
	}
}

func renvo0538Wrap(n int) int {
	if n == 0 {
		return renvo0538Pick(false)
	}
	return renvo0538Wrap(n - 1)
}

func appMain(args []string) int {
	if renvo0538Wrap(2) != 6 {
		print("RENVO-0538 else return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
