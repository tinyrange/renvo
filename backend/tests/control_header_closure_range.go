package main

func rangeHeaderPredicate(f func() bool) bool { return f() }

func appMain() int {
	count := 0
	for rangeHeaderPredicate(func() bool {
		sum := 0
		for _, value := range []int{1, 2} {
			sum += value
		}
		return sum == 3
	}) {
		count++
		break
	}
	if count != 1 {
		return 1
	}
	print("PASS\n")
	return 0
}
