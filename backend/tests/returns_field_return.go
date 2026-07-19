package main

type Renvo0549Item struct {
	value int
}

func renvo0549Field(x Renvo0549Item) int {
	return x.value
}

func appMain(args []string) int {
	item := Renvo0549Item{
		value: 19,
	}
	if renvo0549Field(item) != 19 {
		print("RENVO-0549 field return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
