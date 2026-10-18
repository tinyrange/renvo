package main

const sets = 4096

var sink uint64

func dirty(seed uint64) uint64 {
	var words [256]uint64
	for i := range words {
		words[i] = seed + uint64(i)
	}
	return words[13] ^ words[239]
}
func slotFor(pc uint64) uint64 { return ((pc >> 2) ^ (pc >> 12)) & 4095 }
func publish(pc uint64) {
	blocks := make([]uint64, 16384)
	base, slot := slotFor(pc), uint64(0)
	found := false
	for way := uint64(0); way < 4; way++ {
		i := base + way*sets
		if blocks[i] == 0 {
			slot, found = i, true
			break
		}
	}
	if !found {
		slot = base
	}
	blocks[slot] = pc
	if blocks[slot] != pc {
		print("FAIL\n")
	}
}
func appMain() int {
	sink = dirty(0xffff800080cca9dc)
	publish(0xffff800080cca9dc)
	print("PASS\n")
	return 0
}
