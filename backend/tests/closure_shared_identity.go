package main

type closureIdentityRecord struct {
	value int
}

func closureIdentityCounter(start int) func() int {
	value := start
	return func() int {
		value++
		return value
	}
}

func closureIdentityTypes() bool {
	number := 5
	pointerValue := 8
	pointer := &number
	text := "old"
	values := []int{6}
	record := closureIdentityRecord{value: 7}
	function := func(input int) int { return input + 1 }
	read := func() bool {
		if number != 8 || *pointer != 8 || text != "new" || values[0] != 9 || record.value != 10 {
			return false
		}
		return function(10) == 12
	}
	number = 8
	pointer = &pointerValue
	text = "new"
	values = []int{9}
	record.value = 10
	function = func(input int) int { return input + 2 }
	return read()
}

func closureIdentityInterface() bool {
	var boxed interface{} = 11
	write := func() { boxed = 12 }
	write()
	return boxed.(int) == 12
}

func closureIdentityDeferred(out *int) {
	value := 1
	defer func() { *out = value }()
	value = 2
}

func appMain() int {
	value := 1
	read := func() int { return value }
	write := func(next int) { value = next }
	value = 2
	if read() != 2 {
		return 1
	}
	write(3)
	if value != 3 || read() != 3 {
		return 1
	}
	next := closureIdentityCounter(3)
	if next() != 4 || next() != 5 || !closureIdentityTypes() || !closureIdentityInterface() {
		return 1
	}
	deferred := 0
	closureIdentityDeferred(&deferred)
	if deferred != 2 {
		return 1
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
		return 1
	}
	items := []int{6, 7}
	for _, item := range items {
		if item == 6 {
			first = func() int { return item }
		} else {
			second = func() int { return item }
		}
	}
	if first() != 6 || second() != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
