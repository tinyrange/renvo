package main

type Renvo0624Box struct{ value int }

func appMain(args []string) int {
	box := Renvo0624Box{
		value: int(byte(33)),
	}
	if box.value != 33 {
		print("RENVO-0624 conversion field init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
