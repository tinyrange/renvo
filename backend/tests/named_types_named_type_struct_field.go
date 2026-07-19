package main

type renvo0672Amount int
type renvo0672Box struct {
	value renvo0672Amount
}

func appMain(args []string) int {
	b := renvo0672Box{value: renvo0672Amount(9)}
	if int(b.value) != 9 {
		print("RENVO-0672 named field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
