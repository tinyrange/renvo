package main

var rangeArrayCalls int

func rangeArraySource() [3]int {
	rangeArrayCalls++
	return [3]int{4, 5, 6}
}
func rangeArrayPointer() *[2]int {
	rangeArrayCalls++
	return nil
}
func rangeArrayElement() int {
	rangeArrayCalls++
	return 7
}
func appMain() int {
	var missing *[3]int
	total := 0
	for i := range *missing {
		total += i
	}
	for i, _ := range *missing {
		total += i
	}
	for range missing {
		total++
	}
	if total != 9 {
		return 1
	}
	for i := range rangeArraySource() {
		total += i
	}
	for range rangeArrayPointer() {
		total++
	}
	for range [2]int{rangeArrayElement(), 1} {
		total++
	}
	if rangeArrayCalls != 3 || total != 16 {
		return 2
	}
	for range [1]int{len(rangeArraySource())} {
		total++
	}
	for range [1]int{cap(rangeArrayPointer())} {
		total++
	}
	if rangeArrayCalls != 5 || total != 18 {
		return 4
	}
	values := [3]int{4, 5, 6}
	snapshot := 0
	for i, value := range values {
		snapshot += value
		values[(i+1)%3] = 90
	}
	if snapshot != 15 {
		return 3
	}
	print("PASS\n")
	return 0
}
