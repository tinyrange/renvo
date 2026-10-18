package main

func checkedSlice(values []int, offset int) []int {
	if offset < 0 {
		panic("slice")
	}
	return values[offset:]
}
func checkedText(value string, offset int) string {
	if offset < 0 {
		panic("text")
	}
	return value[offset:]
}
func checkedWide(value uint64) uint64 {
	if value == 0 {
		panic("wide")
	}
	return value ^ 0x123456789abcdef0
}
func checkedValues(values []int, offset int) bool {
	part := checkedSlice(values, offset)
	text := checkedText("prefix-result", 7)
	wide := checkedWide(0x76543210fedcba98)
	return len(part) == 2 && cap(part) == 3 && part[0] == 17 && part[1] == 29 &&
		text == "result" && wide == 0x6460646864606468
}
func checkedDuringPanic(values []int, offset int) (caught, complete bool) {
	defer func() { caught = recover() != nil }()
	defer func() { complete = checkedValues(values, offset) }()
	panic("parent identity")
}
func checkedNewPanic(values []int, offset int) (caught bool) {
	defer func() { caught = recover() != nil }()
	checkedSlice(values, offset)
	return false
}
func appMain() int {
	values := make([]int, 3, 4)
	values[0], values[1], values[2] = 3, 17, 29
	if !checkedValues(values, 1) {
		panic("normal result registers")
	}
	caught, complete := checkedDuringPanic(values, 1)
	if !caught || !complete {
		panic("parent panic result registers")
	}
	if !checkedNewPanic(values, -1) {
		panic("new panic propagation")
	}
	print("PASS\n")
	return 0
}
