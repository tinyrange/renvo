package main

func makeCounter(start int) func() int {
	value := start
	return func() int {
		value++
		return value
	}
}

func main() {
	value := 1
	read := func() int { return value }
	write := func(next int) { value = next }
	value = 2
	if read() != 2 {
		return
	}
	write(3)
	if value != 3 || read() != 3 {
		return
	}
	next := makeCounter(3)
	if next() != 4 || next() != 5 {
		return
	}
	var first func() int
	var second func() int
	for index := 0; index < 2; index++ {
		if index == 0 {
			first = func() int { return index }
		} else {
			second = func() int { return index }
		}
	}
	if first() != 0 || second() != 1 {
		return
	}
	values := []int{6, 7}
	for _, item := range values {
		if item == 6 {
			first = func() int { return item }
		} else {
			second = func() int { return item }
		}
	}
	if first() != 6 || second() != 7 {
		return
	}
	print("PASS\n")
}
