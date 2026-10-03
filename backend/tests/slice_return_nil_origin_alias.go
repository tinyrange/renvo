package main

var sharedNilOrigin []int

func appendNilOrigin(values []int) []int {
	values = append(values, 7)
	sharedNilOrigin = values
	return values
}

func buildNilOrigin() []int {
	var values []int
	values = appendNilOrigin(values)
	return values
}

func appMain() int {
	values := buildNilOrigin()
	sharedNilOrigin[0] = 9
	if values[0] != 9 {
		panic("returned slice lost its backing identity")
	}
	values[0] = 11
	if sharedNilOrigin[0] != 11 {
		panic("returned slice lost its shared storage")
	}
	print("PASS\n")
	return 0
}
