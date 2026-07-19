package main

func renvo0512SumEven(n int, acc int) int {
	if n == 0 {
		return acc
	}
	return renvo0512SumOdd(n-1, acc+n)
}

func renvo0512SumOdd(n int, acc int) int {
	if n == 0 {
		return acc
	}
	return renvo0512SumEven(n-1, acc+n)
}

func appMain(args []string) int {
	value := 0
	goto compute
compute:
	value = renvo0512SumEven(5, 0)
	if value != 15 {
		print("RENVO-0512 mutual accumulator failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
