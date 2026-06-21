package main

type rtg1014Box struct {
	value int
}

func rtg1014Make() (*int, rtg1014Box) {
	x := 5
	return &x, rtg1014Box{value: 9}
}

func appMain(args []string) int {
	p, box := rtg1014Make()
	if *p+box.value != 14 {
		print("RTG-1014 pointer struct returns failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
