package main

type capCounts []int

type capBox struct {
	items []int
}

func capFromCall() []int {
	return make([]int, 1, 4)
}

func appMain() int {
	xs := make([]int, 2, 5)
	ys := xs[:1:3]
	lit := []int{1, 2, 3}
	named := capCounts{4, 5}
	b := capBox{items: make([]int, 0, 6)}
	called := capFromCall()
	if cap(xs) == 5 && cap(ys) == 3 && cap(lit) == 3 && cap(named) == 2 && cap(b.items) == 6 && cap(called) == 4 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
