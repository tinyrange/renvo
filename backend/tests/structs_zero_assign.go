package main

type Renvo0611Box struct{ value int }

func appMain(args []string) int {
	var b Renvo0611Box
	for i := 0; i < 3; i = i + 1 {
		if i == 1 {
			continue
		}
		b.value = b.value + i
	}
	if b.value != 2 {
		print("RENVO-0611 zero assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
