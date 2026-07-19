package main

type renvo1014Box struct {
	value int
}

func renvo1014Make() (*int, renvo1014Box) {
	x := 5
	return &x, renvo1014Box{value: 9}
}

func appMain(args []string) int {
	p, box := renvo1014Make()
	if *p+box.value != 14 {
		print("RENVO-1014 pointer struct returns failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
