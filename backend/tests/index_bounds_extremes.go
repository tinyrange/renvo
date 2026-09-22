package main

func indexedByte(index int) (caught bool) {
	defer func() { caught = recover() != nil }()
	values := []byte{11, 22, 33}
	values[index] = values[index] + 1
	return false
}

func indexedWord(index int) (caught bool) {
	defer func() { caught = recover() != nil }()
	values := []int{11, 22, 33}
	values[index] = values[index] + 1
	return false
}

func indexedString(index int) (caught bool) {
	defer func() { caught = recover() != nil }()
	value := "abc"
	if value[index] == 0 {
		return false
	}
	return false
}

func appMain() int {
	indices := []int{-2147483648, -1, 0, 2, 3, 2147483647}
	for _, index := range indices {
		want := index < 0 || index >= 3
		if indexedByte(index) != want || indexedWord(index) != want || indexedString(index) != want {
			return 1
		}
	}
	values := []int{11, 22, 33}
	bytes := []byte{1, 2, 3}
	for i := 0; i < len(values); i++ {
		values[i] += i
		bytes[i] += byte(i)
	}
	if values[0] != 11 || values[1] != 23 || values[2] != 35 || bytes[0] != 1 || bytes[1] != 3 || bytes[2] != 5 {
		return 2
	}
	print("PASS\n")
	return 0
}
