package main

type renvo0340Box struct{ value int }

func appMain(args []string) int {
	b := renvo0340Box{value: 4}
	b.value += 5
	if b.value != 9 {
		print("RENVO-0340 struct field compound failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
