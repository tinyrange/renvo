package main

func lengthComparisons(value int, data []byte) bool {
	length := len(data)
	capacity := cap(data)
	return value < length && value <= capacity && length > value && capacity >= value
}

func appMain() int {
	data := make([]byte, 3, 7)
	maximum := int(^uint(0) / 2)
	minimum := -maximum - 1
	if !lengthComparisons(minimum, data) || !lengthComparisons(-1, data) || !lengthComparisons(2, data) || lengthComparisons(3, data) || lengthComparisons(maximum, data) {
		panic("immutable length")
	}
	length := len(data)
	length = -1
	if length >= 0 || length < minimum {
		panic("reassigned length")
	}
	aliased := len(data)
	pointer := &(aliased)
	*pointer = minimum
	if aliased >= 0 || aliased > -1 {
		panic("aliased length")
	}
	for remaining := len(data); remaining >= -1; remaining-- {
		if remaining == -1 && !(remaining < 0) {
			panic("mutated loop length")
		}
	}
	for index := 0; index < len(data); index++ {
		if index < 0 || index >= cap(data) {
			panic("bounded counter")
		}
	}
	for index := 0; index < len(data); index++ {
		index = -1
		if index >= 0 {
			panic("authored counter write")
		}
		break
	}
	for index := 0; index < len(data); index++ {
		pointer := &(index)
		*pointer = minimum
		if index >= 0 {
			panic("aliased counter")
		}
		break
	}
	{
		value := len(data)
		if value <= -1 {
			panic("positive scoped length")
		}
	}
	{
		value := minimum
		if value >= 0 {
			panic("reused local slot")
		}
	}
	print("PASS\n")
	return 0
}
