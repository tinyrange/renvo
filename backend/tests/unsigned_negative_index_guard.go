package main

func bounded(values []int, index int) bool {
	if uint(index) >= uint(len(values)) {
		return false
	}
	return values[index] != 0
}
func appMain() int {
	values := []int{1, 2}
	if bounded(values, -1) {
		return 1
	}
	if bounded(values, 2) {
		return 2
	}
	if !bounded(values, 0) {
		return 3
	}
	print("PASS\n")
	return 0
}
