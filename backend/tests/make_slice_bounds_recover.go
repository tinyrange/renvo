package main

type MakeBoundsEmpty struct{}

var makeBoundsOrder int

func makeBoundsLength(value int) int   { makeBoundsOrder = makeBoundsOrder*10 + 1; return value }
func makeBoundsCapacity(value int) int { makeBoundsOrder = makeBoundsOrder*10 + 2; return value }

func makeEmptyBounds(length, capacity int) (caught bool) {
	defer func() { caught = recover() != nil }()
	values := make([]MakeBoundsEmpty, makeBoundsLength(length), makeBoundsCapacity(capacity))
	_ = values
	return
}

func makeIntBounds(length, capacity int) (caught bool) {
	defer func() { caught = recover() != nil }()
	values := make([]int, makeBoundsLength(length), makeBoundsCapacity(capacity))
	_ = values
	return
}

func appMain() int {
	makeBoundsOrder = 0
	if !makeEmptyBounds(-1, 0) || makeBoundsOrder != 12 {
		panic("negative empty length or argument order")
	}
	if !makeEmptyBounds(1, 0) {
		panic("empty length exceeds capacity")
	}
	if !makeEmptyBounds(0, -1) {
		panic("negative empty capacity")
	}
	makeBoundsOrder = 0
	if !makeIntBounds(-1, 0) || makeBoundsOrder != 12 {
		panic("negative int length or argument order")
	}
	if !makeIntBounds(1, 0) {
		panic("int length exceeds capacity")
	}
	if !makeIntBounds(0, -1) {
		panic("negative int capacity")
	}
	if !makeIntBounds(0, int(^uint(0)>>1)) {
		panic("capacity multiplication overflow")
	}
	if makeEmptyBounds(0, 0) || makeEmptyBounds(1, 2) || makeIntBounds(0, 0) || makeIntBounds(1, 2) {
		panic("valid make panicked")
	}
	print("PASS\n")
	return 0
}
