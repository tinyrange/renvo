package main

type Renvo0539Box struct {
	value int
}

func renvo0539Pick(n int) int {
	if n < 0 {
		return 1
	} else if n == 0 {
		return 2
	}
	return 3
}

func appMain(args []string) int {
	b := Renvo0539Box{value: renvo0539Pick(0)}
	if b.value != 2 {
		print("RENVO-0539 else-if return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
