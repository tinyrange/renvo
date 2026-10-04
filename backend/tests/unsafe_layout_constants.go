package main

type LayoutConstantsRecord struct {
	First byte
	Small uint16
	Last  int
}

var layoutConstantsValue LayoutConstantsRecord
var layoutConstantsCalls int

const layoutConstantsWord = Sizeof(int(0))
const layoutConstantsEight = Sizeof(uint64(0))
const layoutConstantsSize = Sizeof(layoutConstantsValue)
const layoutConstantsAlign = Alignof(layoutConstantsValue)
const layoutConstantsOffset = Offsetof(layoutConstantsValue.Last)

type LayoutConstantsArray [Sizeof(uint64(0))]byte

func layoutConstantsTick() LayoutConstantsRecord {
	layoutConstantsCalls++
	return layoutConstantsValue
}

func appMain(args []string) int {
	const local = Sizeof(uint64(0))
	const sum = Sizeof(int(0)) + Alignof(int(0))
	const offset = Offsetof(layoutConstantsValue.Last)
	const unevaluated = Sizeof(layoutConstantsTick())
	var array LayoutConstantsArray
	var localArray [local]byte
	if layoutConstantsEight != 8 || local != 8 || len(array) != 8 || len(localArray) != 8 {
		panic("scalar layout constant or array bound")
	}
	if layoutConstantsWord != Sizeof(int(0)) || sum != Sizeof(int(0))+Alignof(int(0)) {
		panic("target word layout constant")
	}
	if layoutConstantsSize != Sizeof(layoutConstantsValue) || unevaluated != Sizeof(layoutConstantsValue) {
		panic("aggregate layout constant")
	}
	if layoutConstantsAlign != Alignof(layoutConstantsValue) || layoutConstantsOffset != Offsetof(layoutConstantsValue.Last) || offset != layoutConstantsOffset {
		panic("aggregate alignment or offset constant")
	}
	if layoutConstantsCalls != 0 {
		panic("layout operand evaluated")
	}
	print("PASS\n")
	return 0
}
